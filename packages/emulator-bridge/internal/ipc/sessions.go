package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/anthropics/yepanywhere/emulator-bridge/internal/emulator"
	"github.com/anthropics/yepanywhere/emulator-bridge/internal/encoder"
	"github.com/anthropics/yepanywhere/emulator-bridge/internal/stream"
)

// SessionStartOptions are the options for starting an emulator streaming session.
type SessionStartOptions struct {
	MaxFPS   int `json:"maxFps"`
	MaxWidth int `json:"maxWidth"`
}

// streamSession holds the state for a single active emulator streaming session.
type streamSession struct {
	sessionID   string
	emulatorID  string
	client      *emulator.Client
	frameSource *emulator.FrameSource
	enc         *encoder.H264Encoder
	peer        *stream.PeerSession
	input       *stream.InputHandler
	cancel      context.CancelFunc
	targetW     int
	targetH     int
}

// SessionManager manages multiple concurrent emulator streaming sessions.
type SessionManager struct {
	mu          sync.Mutex
	sessions    map[string]*streamSession
	stunServers []string
	sendMsg     func(msg []byte) // send JSON to the Yep server WebSocket
}

// NewSessionManager creates a session manager.
func NewSessionManager(stunServers []string, sendMsg func(msg []byte)) *SessionManager {
	return &SessionManager{
		sessions:    make(map[string]*streamSession),
		stunServers: stunServers,
		sendMsg:     sendMsg,
	}
}

// StartSession creates a new streaming session for the given emulator.
func (sm *SessionManager) StartSession(sessionID, emulatorID string, opts SessionStartOptions) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Close existing session with same ID.
	if existing, ok := sm.sessions[sessionID]; ok {
		sm.closeSessionLocked(existing)
	}

	sm.sendState(sessionID, "connecting", "")

	// Defaults.
	maxWidth := opts.MaxWidth
	if maxWidth <= 0 {
		maxWidth = 540
	}
	maxFPS := opts.MaxFPS
	if maxFPS <= 0 {
		maxFPS = 30
	}

	// Connect to emulator gRPC.
	grpcAddr := GRPCAddr(emulatorID)
	log.Printf("[session %s] connecting to emulator %s at %s", sessionID, emulatorID, grpcAddr)

	client, err := emulator.NewClient(grpcAddr)
	if err != nil {
		sm.sendState(sessionID, "failed", fmt.Sprintf("emulator connect: %v", err))
		return fmt.Errorf("connecting to emulator: %w", err)
	}

	srcW, srcH := client.ScreenSize()
	targetW, targetH := encoder.ComputeTargetSize(int(srcW), int(srcH), maxWidth)
	log.Printf("[session %s] screen %dx%d → encoding %dx%d", sessionID, srcW, srcH, targetW, targetH)

	h264Enc, err := encoder.NewH264Encoder(targetW, targetH, maxFPS)
	if err != nil {
		client.Close()
		sm.sendState(sessionID, "failed", fmt.Sprintf("encoder: %v", err))
		return fmt.Errorf("creating encoder: %w", err)
	}

	frameSource := emulator.NewFrameSource(client)
	inputHandler := stream.NewInputHandler(client)

	// Create WebRTC peer with trickle ICE.
	onICE := func(c *stream.ICECandidateJSON) {
		sm.sendICE(sessionID, c)
	}
	peer, err := stream.NewPeerSession(sm.stunServers, inputHandler.HandleMessage, onICE)
	if err != nil {
		frameSource.Stop()
		h264Enc.Close()
		client.Close()
		sm.sendState(sessionID, "failed", fmt.Sprintf("peer: %v", err))
		return fmt.Errorf("creating peer: %w", err)
	}

	sdp, err := peer.CreateOffer()
	if err != nil {
		peer.Close()
		frameSource.Stop()
		h264Enc.Close()
		client.Close()
		sm.sendState(sessionID, "failed", fmt.Sprintf("offer: %v", err))
		return fmt.Errorf("creating offer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sess := &streamSession{
		sessionID:   sessionID,
		emulatorID:  emulatorID,
		client:      client,
		frameSource: frameSource,
		enc:         h264Enc,
		peer:        peer,
		input:       inputHandler,
		cancel:      cancel,
		targetW:     targetW,
		targetH:     targetH,
	}
	sm.sessions[sessionID] = sess

	// Monitor peer close.
	go func() {
		select {
		case <-peer.Done():
			sm.mu.Lock()
			if s, ok := sm.sessions[sessionID]; ok && s == sess {
				sm.closeSessionLocked(sess)
				delete(sm.sessions, sessionID)
			}
			sm.mu.Unlock()
			sm.sendState(sessionID, "disconnected", "")
		case <-ctx.Done():
		}
	}()

	// Send the offer to the Yep server.
	sm.sendOffer(sessionID, sdp)

	return nil
}

// HandleAnswer processes an SDP answer for a session and starts the encoding pipeline.
func (sm *SessionManager) HandleAnswer(sessionID, sdp string) error {
	sm.mu.Lock()
	sess, ok := sm.sessions[sessionID]
	sm.mu.Unlock()

	if !ok {
		return fmt.Errorf("no session %s", sessionID)
	}

	if err := sess.peer.SetAnswer(sdp); err != nil {
		return fmt.Errorf("setting answer: %w", err)
	}

	// Start encoding pipeline.
	go sm.runPipeline(sess)

	sm.sendState(sessionID, "connected", "")
	return nil
}

// HandleICE adds a remote ICE candidate to a session.
func (sm *SessionManager) HandleICE(sessionID string, candidateJSON json.RawMessage) error {
	sm.mu.Lock()
	sess, ok := sm.sessions[sessionID]
	sm.mu.Unlock()

	if !ok {
		return fmt.Errorf("no session %s", sessionID)
	}

	if string(candidateJSON) == "null" {
		// End-of-candidates signal; nothing to do on the Pion side.
		return nil
	}

	return sess.peer.AddICECandidate(candidateJSON)
}

// StopSession tears down a streaming session.
func (sm *SessionManager) StopSession(sessionID string) {
	sm.mu.Lock()
	sess, ok := sm.sessions[sessionID]
	if ok {
		sm.closeSessionLocked(sess)
		delete(sm.sessions, sessionID)
	}
	sm.mu.Unlock()

	if ok {
		sm.sendState(sessionID, "disconnected", "")
	}
}

// CloseAll tears down all sessions.
func (sm *SessionManager) CloseAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for id, sess := range sm.sessions {
		sm.closeSessionLocked(sess)
		delete(sm.sessions, id)
	}
}

func (sm *SessionManager) closeSessionLocked(sess *streamSession) {
	sess.cancel()
	sess.peer.Close()
	sess.frameSource.Stop()
	sess.enc.Close()
	sess.client.Close()
	log.Printf("[session %s] closed", sess.sessionID)
}

func (sm *SessionManager) runPipeline(sess *streamSession) {
	// Wait for WebRTC connection before encoding frames.
	// Frames encoded before the connection is ready are silently dropped by Pion.
	log.Printf("[session %s] pipeline waiting for WebRTC connection", sess.sessionID)
	select {
	case <-sess.peer.Connected():
		log.Printf("[session %s] WebRTC connected, starting pipeline", sess.sessionID)
	case <-sess.peer.Done():
		log.Printf("[session %s] peer closed before connecting", sess.sessionID)
		return
	case <-time.After(30 * time.Second):
		log.Printf("[session %s] WebRTC connection timed out", sess.sessionID)
		return
	}

	id, frames := sess.frameSource.Subscribe()
	defer sess.frameSource.Unsubscribe(id)

	log.Printf("[session %s] pipeline started", sess.sessionID)
	defer log.Printf("[session %s] pipeline stopped", sess.sessionID)

	var lastTime time.Time

	for {
		select {
		case <-sess.peer.Done():
			return
		case frame, ok := <-frames:
			if !ok {
				return
			}

			scaled := encoder.Scale(
				frame.Data,
				int(frame.Width), int(frame.Height),
				sess.targetW, sess.targetH,
			)

			y, cb, cr := encoder.RGBToI420(scaled, sess.targetW, sess.targetH)

			nals, err := sess.enc.Encode(y, cb, cr)
			if err != nil {
				log.Printf("[session %s] encode error: %v", sess.sessionID, err)
				continue
			}
			if nals == nil {
				continue
			}

			now := time.Now()
			duration := time.Second / 30
			if !lastTime.IsZero() {
				duration = now.Sub(lastTime)
			}
			lastTime = now

			if err := sess.peer.WriteVideoSample(nals, duration); err != nil {
				log.Printf("[session %s] write error: %v", sess.sessionID, err)
				return
			}
		}
	}
}

// sendOffer sends a WebRTC offer to the Yep server.
func (sm *SessionManager) sendOffer(sessionID, sdp string) {
	msg, _ := json.Marshal(map[string]string{
		"type":      "webrtc.offer",
		"sessionId": sessionID,
		"sdp":       sdp,
	})
	sm.sendMsg(msg)
}

// sendICE sends an ICE candidate to the Yep server.
func (sm *SessionManager) sendICE(sessionID string, candidate *stream.ICECandidateJSON) {
	m := map[string]interface{}{
		"type":      "webrtc.ice",
		"sessionId": sessionID,
	}
	if candidate == nil {
		m["candidate"] = nil
	} else {
		m["candidate"] = candidate
	}
	msg, _ := json.Marshal(m)
	sm.sendMsg(msg)
}

// sendState sends a session state change to the Yep server.
func (sm *SessionManager) sendState(sessionID, state, errMsg string) {
	m := map[string]string{
		"type":      "session.state",
		"sessionId": sessionID,
		"state":     state,
	}
	if errMsg != "" {
		m["error"] = errMsg
	}
	msg, _ := json.Marshal(m)
	sm.sendMsg(msg)
}
