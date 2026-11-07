package rex

import (
	"encoding/base64"
	"strings"
	"time"
)

// AgentStatus is what a coding agent in a session last said it does, by
// OSC 7501: ESC ] 7501 ; state=working:app=claude-code:… ST, its keys
// joined by colons, title and msg in base64.
type AgentStatus struct {
	// State is working, blocked, done, idle or error.
	State string `json:"state"`
	App   string `json:"app,omitempty"`
	// Kind is what a blocked agent waits for: permission, question,
	// auth…
	Kind    string    `json:"kind,omitempty"`
	Msg     string    `json:"msg,omitempty"`
	Updated time.Time `json:"updated"`
}

// statusReply answers the probe an agent sends as it starts, ESC ] 7501 ;
// ? ST: agents report their state only to terminals that answer it.
var statusReply = []byte("\x1b]7501;?\x1b\\")

// maxStatus bounds the OSC the scanner keeps: longer ones are not 7501's.
const maxStatus = 4096

// statusScanner finds the OSC 7501 sequences in what a session prints,
// whichever reads they span.
type statusScanner struct {
	state int // scanGround, …
	buf   []byte
	skip  bool // an OSC of another code, or too long
}

const (
	scanGround = iota
	scanEsc    // after ESC
	scanOSC    // in an OSC
	scanOSCEsc // after ESC in an OSC: ST, or another sequence cutting it
)

const statusPrefix = "7501;"

// scan feeds what a session printed, calling found with the payload of
// each OSC 7501 that ends in it.
func (sc *statusScanner) scan(p []byte, found func(payload string)) {
	for _, b := range p {
		switch sc.state {
		case scanGround:
			if b == 0x1b {
				sc.state = scanEsc
			}
		case scanEsc:
			sc.escape(b)
		case scanOSC:
			switch b {
			case 0x07:
				sc.end(found)
			case 0x1b:
				sc.state = scanOSCEsc
			case 0x18, 0x1a: // CAN and SUB cancel it
				sc.state = scanGround
			default:
				sc.add(b)
			}
		case scanOSCEsc:
			if b == '\\' {
				sc.end(found)
			} else {
				sc.escape(b)
			}
		}
	}
}

// escape takes the byte after an ESC.
func (sc *statusScanner) escape(b byte) {
	switch b {
	case ']':
		sc.state, sc.buf, sc.skip = scanOSC, sc.buf[:0], false
	case 0x1b:
		sc.state = scanEsc
	default:
		sc.state = scanGround
	}
}

func (sc *statusScanner) add(b byte) {
	if sc.skip {
		return
	}
	if n := len(sc.buf); n >= maxStatus || n < len(statusPrefix) && b != statusPrefix[n] {
		sc.skip = true
		return
	}
	sc.buf = append(sc.buf, b)
}

func (sc *statusScanner) end(found func(string)) {
	sc.state = scanGround
	if !sc.skip && len(sc.buf) >= len(statusPrefix) {
		found(string(sc.buf[len(statusPrefix):]))
	}
}

// parseStatus reads the payload of an OSC 7501 that is not a probe. It
// returns nil for state=clear, and ok false for the items of a session
// other than the agent itself, as its subagents, which carry an id.
func parseStatus(payload string, now time.Time) (st *AgentStatus, ok bool) {
	st = &AgentStatus{Updated: now}
	for _, kv := range strings.Split(payload, ":") {
		k, v, _ := strings.Cut(kv, "=")
		switch k {
		case "id":
			if v != "" {
				return nil, false
			}
		case "state":
			st.State = v
		case "app":
			st.App = v
		case "kind":
			st.Kind = v
		case "msg":
			st.Msg = v
			if b, err := base64.StdEncoding.DecodeString(v); err == nil {
				st.Msg = string(b)
			}
		}
	}
	switch st.State {
	case "":
		return nil, false
	case "clear":
		return nil, true
	}
	return st, true
}
