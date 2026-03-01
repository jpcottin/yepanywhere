package stream

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

// ICECandidateJSON is the JSON-serializable form of an ICE candidate.
type ICECandidateJSON struct {
	Candidate        string  `json:"candidate"`
	SDPMid           *string `json:"sdpMid,omitempty"`
	SDPMLineIndex    *uint16 `json:"sdpMLineIndex,omitempty"`
	UsernameFragment *string `json:"usernameFragment,omitempty"`
}

// PeerSession represents one WebRTC connection to a browser.
type PeerSession struct {
	pc          *webrtc.PeerConnection
	videoTrack  *webrtc.TrackLocalStaticSample
	dataChannel *webrtc.DataChannel
	onInput     func(msg []byte)
	closed      chan struct{}
	connected   chan struct{} // closed when ICE reaches "connected" state
}

// NewPeerSession creates a PeerConnection with an h264 video track and a "control" DataChannel.
// onICE is called for each trickle ICE candidate (nil candidate means gathering complete).
func NewPeerSession(stunServers []string, onInput func(msg []byte), onICE func(*ICECandidateJSON)) (*PeerSession, error) {
	iceServers := []webrtc.ICEServer{}
	if len(stunServers) > 0 {
		iceServers = append(iceServers, webrtc.ICEServer{URLs: stunServers})
	}

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: iceServers,
	})
	if err != nil {
		return nil, fmt.Errorf("creating peer connection: %w", err)
	}

	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		"video", "emulator",
	)
	if err != nil {
		pc.Close()
		return nil, fmt.Errorf("creating video track: %w", err)
	}

	if _, err := pc.AddTrack(videoTrack); err != nil {
		pc.Close()
		return nil, fmt.Errorf("adding video track: %w", err)
	}

	dc, err := pc.CreateDataChannel("control", nil)
	if err != nil {
		pc.Close()
		return nil, fmt.Errorf("creating data channel: %w", err)
	}

	ps := &PeerSession{
		pc:          pc,
		videoTrack:  videoTrack,
		dataChannel: dc,
		onInput:     onInput,
		closed:      make(chan struct{}),
		connected:   make(chan struct{}),
	}

	dc.OnOpen(func() {
		log.Printf("DataChannel '%s' opened", dc.Label())
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if ps.onInput != nil {
			ps.onInput(msg.Data)
		}
	})

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE connection state: %s", state.String())
		switch state {
		case webrtc.ICEConnectionStateConnected, webrtc.ICEConnectionStateCompleted:
			select {
			case <-ps.connected:
			default:
				close(ps.connected)
			}
		case webrtc.ICEConnectionStateFailed,
			webrtc.ICEConnectionStateDisconnected,
			webrtc.ICEConnectionStateClosed:
			select {
			case <-ps.closed:
			default:
				close(ps.closed)
			}
		}
	})

	// Trickle ICE: emit candidates as they are discovered.
	if onICE != nil {
		pc.OnICECandidate(func(c *webrtc.ICECandidate) {
			if c == nil {
				// Gathering complete.
				onICE(nil)
				return
			}
			init := c.ToJSON()
			onICE(&ICECandidateJSON{
				Candidate:        init.Candidate,
				SDPMid:           init.SDPMid,
				SDPMLineIndex:    init.SDPMLineIndex,
				UsernameFragment: init.UsernameFragment,
			})
		})
	}

	return ps, nil
}

// CreateOffer creates an SDP offer and returns it immediately (without waiting for ICE gathering).
// ICE candidates are delivered via the onICE callback passed to NewPeerSession.
func (ps *PeerSession) CreateOffer() (string, error) {
	offer, err := ps.pc.CreateOffer(nil)
	if err != nil {
		return "", fmt.Errorf("creating offer: %w", err)
	}

	if err := ps.pc.SetLocalDescription(offer); err != nil {
		return "", fmt.Errorf("setting local description: %w", err)
	}

	return offer.SDP, nil
}

// CreateOfferGathered creates an SDP offer and blocks until ICE gathering is complete.
// Returns the SDP with all candidates embedded. Used for standalone (non-IPC) mode.
func (ps *PeerSession) CreateOfferGathered() (string, error) {
	offer, err := ps.pc.CreateOffer(nil)
	if err != nil {
		return "", fmt.Errorf("creating offer: %w", err)
	}

	gatherComplete := webrtc.GatheringCompletePromise(ps.pc)

	if err := ps.pc.SetLocalDescription(offer); err != nil {
		return "", fmt.Errorf("setting local description: %w", err)
	}

	select {
	case <-gatherComplete:
	case <-time.After(10 * time.Second):
		return "", fmt.Errorf("ICE gathering timed out")
	}

	return ps.pc.LocalDescription().SDP, nil
}

// SetAnswer sets the remote SDP answer from the browser.
func (ps *PeerSession) SetAnswer(sdp string) error {
	return ps.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	})
}

// AddICECandidate adds a remote ICE candidate (trickle ICE).
func (ps *PeerSession) AddICECandidate(candidateJSON []byte) error {
	var candidate ICECandidateJSON
	if err := json.Unmarshal(candidateJSON, &candidate); err != nil {
		return fmt.Errorf("parsing ICE candidate: %w", err)
	}
	return ps.pc.AddICECandidate(webrtc.ICECandidateInit{
		Candidate:        candidate.Candidate,
		SDPMid:           candidate.SDPMid,
		SDPMLineIndex:    candidate.SDPMLineIndex,
		UsernameFragment: candidate.UsernameFragment,
	})
}

// WriteVideoSample sends h264 NAL data to the video track.
func (ps *PeerSession) WriteVideoSample(data []byte, duration time.Duration) error {
	return ps.videoTrack.WriteSample(media.Sample{
		Data:     data,
		Duration: duration,
	})
}

// Close tears down the PeerConnection.
func (ps *PeerSession) Close() error {
	select {
	case <-ps.closed:
	default:
		close(ps.closed)
	}
	return ps.pc.Close()
}

// Connected returns a channel that is closed when the ICE connection is established.
func (ps *PeerSession) Connected() <-chan struct{} {
	return ps.connected
}

// Done returns a channel that is closed when the peer disconnects.
func (ps *PeerSession) Done() <-chan struct{} {
	return ps.closed
}
