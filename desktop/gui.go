package main

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/ngbcoder/baton/desktop/protocol"

	"github.com/skip2/go-qrcode"
)

// ---------------------------------------------------------------------------
// Palette: slate base with an indigo accent. Cards sit one step above the
// page and are outlined, so depth comes from borders rather than shadows.
// ---------------------------------------------------------------------------

var (
	colBg      = color.NRGBA{R: 2, G: 6, B: 23, A: 255}      // slate-950 (window, sidebar, log)
	colContent = color.NRGBA{R: 6, G: 11, B: 30, A: 255}     // page behind the cards
	colCard    = color.NRGBA{R: 15, G: 23, B: 42, A: 255}    // slate-900
	colBorder  = color.NRGBA{R: 30, G: 41, B: 59, A: 255}    // slate-800
	colBorder2 = color.NRGBA{R: 51, G: 65, B: 85, A: 255}    // slate-700
	colFg      = color.NRGBA{R: 241, G: 245, B: 249, A: 255} // slate-100
	colMuted   = color.NRGBA{R: 148, G: 163, B: 184, A: 255} // slate-400
	colDim     = color.NRGBA{R: 100, G: 116, B: 139, A: 255} // slate-500

	colPrimary    = color.NRGBA{R: 99, G: 102, B: 241, A: 255}  // indigo-500
	colPrimarySft = color.NRGBA{R: 99, G: 102, B: 241, A: 38}   // indigo wash
	colIndigoText = color.NRGBA{R: 165, G: 180, B: 252, A: 255} // indigo-300
	colIndigoDeep = color.NRGBA{R: 30, G: 27, B: 75, A: 255}    // indigo-950

	colGreen     = color.NRGBA{R: 16, G: 185, B: 129, A: 255}
	colGreenText = color.NRGBA{R: 52, G: 211, B: 153, A: 255}
	colGreenSft  = color.NRGBA{R: 16, G: 185, B: 129, A: 34}
	colYellow    = color.NRGBA{R: 234, G: 179, B: 8, A: 255}
	colRed       = color.NRGBA{R: 244, G: 63, B: 94, A: 255}
	colRedText   = color.NRGBA{R: 253, G: 164, B: 175, A: 255}
	colRedDeep   = color.NRGBA{R: 76, G: 5, B: 25, A: 255}

	colWhite = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	colQRInk = color.NRGBA{R: 2, G: 6, B: 23, A: 255}
)

// Flip this if a phone can't scan the drawn QR code (some ASCII renderers
// print light modules as blocks).
const qrInvert = false

// ---------------------------------------------------------------------------
// Mock Bluetooth + Audio
// ---------------------------------------------------------------------------

type mockBluetooth struct {
	mu      sync.Mutex
	devices []protocol.BtDevice
	conn    map[string]bool
}

func newMockBluetooth() *mockBluetooth {
	return &mockBluetooth{
		devices: []protocol.BtDevice{},
		conn:    map[string]bool{},
	}
}

func (m *mockBluetooth) PairedDevices() ([]protocol.BtDevice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]protocol.BtDevice, len(m.devices))
	copy(out, m.devices)
	return out, nil
}

func (m *mockBluetooth) IsConnected(addr string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.conn[addr], nil
}

func (m *mockBluetooth) Connect(addr string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conn[addr] = true
	fmt.Printf("[bt] connect %s\n", addr)
	return nil
}

func (m *mockBluetooth) Disconnect(addr string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conn[addr] = false
	fmt.Printf("[bt] disconnect %s\n", addr)
	return nil
}

func (m *mockBluetooth) IsMultipointConnected(addr string) (bool, error) {
	return addr == "74:A3:4A:88:99:BC", nil
}

type mockAudio struct{}

func (m *mockAudio) SetDefaultOutput(addr string) error {
	fmt.Printf("[audio] set default output -> %s\n", addr)
	return nil
}

func (m *mockAudio) PausePlayback() error {
	fmt.Println("[audio] pause playback")
	return nil
}

// ---------------------------------------------------------------------------
// Protocol log
// ---------------------------------------------------------------------------

type protoLog struct {
	mu    sync.Mutex
	lines []string
	max   int
}

func newProtoLog(max int) *protoLog { return &protoLog{max: max} }

func (p *protoLog) Add(s string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	ts := time.Now().Format("15:04:05.000")
	p.lines = append(p.lines, fmt.Sprintf("%s  %s", ts, s))
	if len(p.lines) > p.max {
		p.lines = p.lines[len(p.lines)-p.max:]
	}
}

func (p *protoLog) Lines() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.lines))
	copy(out, p.lines)
	return out
}

func (p *protoLog) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lines = nil
}

// ---------------------------------------------------------------------------
// Shared UI state. Everything the background goroutines touch lives here,
// behind one mutex; the frame loop reads a snapshot.
// ---------------------------------------------------------------------------

const ownerThisPC = "This computer"

var stateOrder = []string{"IDLE", "REQUESTED", "PEER_READY", "RELEASING", "RELEASED", "CONNECTING", "ACTIVE"}

func stepIndex(status string) int {
	for i, s := range stateOrder {
		if s == status {
			return i
		}
	}
	return -1
}

func stateLabel(status string) string {
	switch status {
	case "IDLE":
		return "Idle"
	case "REQUESTED":
		return "Requested"
	case "PEER_READY":
		return "Peer ready"
	case "RELEASING":
		return "Releasing"
	case "RELEASED":
		return "Released"
	case "CONNECTING":
		return "Connecting"
	case "ACTIVE":
		return "Active"
	case "ROLLBACK":
		return "Rolled back"
	}
	return status
}

type uiData struct {
	status     string
	statusCol  color.NRGBA
	step       int
	failed     bool
	owner      string
	peer       string
	peerOnline bool
	pairing    string
	code       string
	qr         [][]bool
	device     string
	addr       string
}

type uiState struct {
	mu       sync.Mutex
	d        uiData
	inv      func()
	peerConn *protocol.TransportConnection
	qrLn     any
}

func (s *uiState) get() uiData {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.d
}

func (s *uiState) update(f func(*uiData)) {
	s.mu.Lock()
	f(&s.d)
	s.mu.Unlock()
	if s.inv != nil {
		s.inv()
	}
}

func (s *uiState) getPeer() *protocol.TransportConnection {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.peerConn
}

func (s *uiState) setPeer(c *protocol.TransportConnection) {
	s.mu.Lock()
	s.peerConn = c
	s.mu.Unlock()
}

// replaceQRListener closes the previous pairing listener (if it can be
// closed) and remembers the new one.
func (s *uiState) replaceQRListener(next any) {
	s.mu.Lock()
	prev := s.qrLn
	s.qrLn = next
	s.mu.Unlock()
	if c, ok := prev.(io.Closer); ok && c != nil {
		_ = c.Close()
	}
}

func (s *uiState) setHandoff(state protocol.HandoffState) {
	s.update(func(d *uiData) {
		name := strings.ToUpper(string(state))
		d.status = name
		d.failed = false
		if i := stepIndex(name); i >= 0 {
			d.step = i
		}
		switch state {
		case protocol.StateActive:
			d.statusCol = colGreen
		case protocol.StateRollback:
			d.statusCol = colRed
			d.status = "ROLLBACK"
			d.failed = true
		case protocol.StateRequested, protocol.StatePeerReady, protocol.StateReleasing, protocol.StateReleased, protocol.StateConnecting:
			d.statusCol = colYellow
		default:
			d.statusCol = colMuted
		}
	})
}

// ---------------------------------------------------------------------------
// Controls
// ---------------------------------------------------------------------------

type ctrls struct {
	navs                                                    [4]widget.Clickable
	pull, sendPhone, phoneHandoff, rollback, refresh        widget.Clickable
	clearLog, genQR, revoke                                 widget.Clickable
	autoPull, callRoute                                     widget.Bool
	pages                                                   [3]widget.List
	logList                                                 widget.List
}

func newCtrls() *ctrls {
	c := &ctrls{}
	c.autoPull.Value = true
	c.callRoute.Value = true
	for i := range c.pages {
		c.pages[i].Axis = layout.Vertical
	}
	c.logList.Axis = layout.Vertical
	c.logList.ScrollToEnd = true
	return c
}

// ---------------------------------------------------------------------------
// Drawing helpers
// ---------------------------------------------------------------------------

func withAlpha(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

func lift(c color.NRGBA, d int) color.NRGBA {
	add := func(v uint8) uint8 {
		n := int(v) + d
		if n > 255 {
			n = 255
		}
		return uint8(n)
	}
	return color.NRGBA{R: add(c.R), G: add(c.G), B: add(c.B), A: c.A}
}

func fillRect(gtx layout.Context, col color.NRGBA) {
	paint.FillShape(gtx.Ops, col, clip.Rect{Max: gtx.Constraints.Min}.Op())
}

func roundedRect(gtx layout.Context, col color.NRGBA, r int) layout.Dimensions {
	sz := gtx.Constraints.Min
	paint.FillShape(gtx.Ops, col, clip.UniformRRect(image.Rectangle{Max: sz}, r).Op(gtx.Ops))
	return layout.Dimensions{Size: sz}
}

// pillRect fills the constraint box with fully rounded ends.
func pillRect(gtx layout.Context, col color.NRGBA) layout.Dimensions {
	return roundedRect(gtx, col, gtx.Constraints.Min.Y/2)
}

func hline(gtx layout.Context, col color.NRGBA) layout.Dimensions {
	sz := image.Pt(gtx.Constraints.Max.X, gtx.Dp(1))
	paint.FillShape(gtx.Ops, col, clip.Rect{Max: sz}.Op())
	return layout.Dimensions{Size: sz}
}

func vline(gtx layout.Context, col color.NRGBA) layout.Dimensions {
	sz := image.Pt(gtx.Dp(1), gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, col, clip.Rect{Max: sz}.Op())
	return layout.Dimensions{Size: sz}
}

func disc(gtx layout.Context, center image.Point, d int, col color.NRGBA) {
	defer op.Offset(center.Sub(image.Pt(d/2, d/2))).Push(gtx.Ops).Pop()
	paint.FillShape(gtx.Ops, col, clip.Ellipse{Max: image.Pt(d, d)}.Op(gtx.Ops))
}

func dot(gtx layout.Context, col color.NRGBA, d unit.Dp) layout.Dimensions {
	sz := gtx.Dp(d)
	disc(gtx, image.Pt(sz/2, sz/2), sz, col)
	return layout.Dimensions{Size: image.Pt(sz, sz)}
}

func empty(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }

func sp(h float32) layout.FlexChild  { return layout.Rigid(layout.Spacer{Height: unit.Dp(h)}.Layout) }
func spx(w float32) layout.FlexChild { return layout.Rigid(layout.Spacer{Width: unit.Dp(w)}.Layout) }

func label(th *material.Theme, text string, size float32, col color.NRGBA, weight font.Weight) material.LabelStyle {
	l := material.Label(th, unit.Sp(size), text)
	l.Color = col
	l.Font.Weight = weight
	return l
}

func monoLabel(th *material.Theme, text string, size float32, col color.NRGBA) material.LabelStyle {
	l := label(th, text, size, col, font.Normal)
	l.Font.Typeface = "Go Mono"
	return l
}

// surface draws an outlined, filled panel that spans the available width.
func surface(gtx layout.Context, fill, stroke color.NRGBA, radius, pad float32, w layout.Widget) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	r := unit.Dp(radius)
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return roundedRect(gtx, fill, gtx.Dp(r))
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return widget.Border{Color: stroke, CornerRadius: r, Width: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(pad)).Layout(gtx, w)
			})
		}),
	)
}

func chip(gtx layout.Context, th *material.Theme, text string, fg, bg color.NRGBA, withDot bool) layout.Dimensions {
	return layout.Stack{Alignment: layout.W}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return pillRect(gtx, bg)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 4, Bottom: 4, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !withDot {
							return layout.Dimensions{}
						}
						return dot(gtx, fg, 6)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !withDot {
							return layout.Dimensions{}
						}
						return layout.Spacer{Width: unit.Dp(6)}.Layout(gtx)
					}),
					layout.Rigid(label(th, text, 11, fg, font.Medium).Layout),
				)
			})
		}),
	)
}

type btnKind int

const (
	btnPrimary btnKind = iota
	btnSecondary
	btnDanger
)

func button(gtx layout.Context, th *material.Theme, c *widget.Clickable, text string, kind btnKind, small, fill bool) layout.Dimensions {
	bg, fg := colPrimary, colWhite
	switch kind {
	case btnSecondary:
		bg, fg = colBorder, colFg
	case btnDanger:
		bg, fg = colRedDeep, colRedText
	}
	if c.Hovered() {
		bg = lift(bg, 16)
	}
	size, padV, padH := float32(13), float32(10), float32(20)
	if small {
		size, padV, padH = 11, 6, 14
	}
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if fill {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
		}
		return layout.Stack{Alignment: layout.Center}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				return pillRect(gtx, bg)
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(padV), Bottom: unit.Dp(padV), Left: unit.Dp(padH), Right: unit.Dp(padH)}.Layout(gtx,
					label(th, text, size, fg, font.Medium).Layout)
			}),
		)
	})
}

func logoMark(gtx layout.Context, size unit.Dp) layout.Dimensions {
	sz := gtx.Dp(size)
	c := image.Pt(sz/2, sz/2)
	disc(gtx, c, sz, colPrimary)
	disc(gtx, c, sz*11/20, colBg)
	disc(gtx, c, sz*5/20, colIndigoText)
	return layout.Dimensions{Size: image.Pt(sz, sz)}
}

func glyph(gtx layout.Context, size unit.Dp) layout.Dimensions {
	sz := gtx.Dp(size)
	c := image.Pt(sz/2, sz/2)
	disc(gtx, c, sz, colIndigoDeep)
	disc(gtx, c, sz*1/2, colPrimary)
	disc(gtx, c, sz*3/10, colIndigoDeep)
	disc(gtx, c, sz*3/20, colIndigoText)
	return layout.Dimensions{Size: image.Pt(sz, sz)}
}

// ---------------------------------------------------------------------------
// QR
// ---------------------------------------------------------------------------

// parseQR turns the ASCII rendering into a module matrix. It understands
// half-block output (▀ ▄ █) and falls back to two characters per module.
func parseQR(ascii string) [][]bool {
	lines := strings.Split(strings.TrimRight(ascii, "\r\n"), "\n")
	half := strings.ContainsAny(ascii, "▀▄")
	var rows [][]bool
	if half {
		for _, ln := range lines {
			rs := []rune(ln)
			top := make([]bool, len(rs))
			bot := make([]bool, len(rs))
			for i, r := range rs {
				switch r {
				case '█':
					top[i], bot[i] = true, true
				case '▀':
					top[i] = true
				case '▄':
					bot[i] = true
				}
			}
			rows = append(rows, top, bot)
		}
	} else {
		for _, ln := range lines {
			rs := []rune(ln)
			row := make([]bool, 0, len(rs)/2+1)
			for i := 0; i < len(rs); i += 2 {
				row = append(row, rs[i] != ' ' && rs[i] != '\u00a0')
			}
			rows = append(rows, row)
		}
	}
	if qrInvert {
		for _, r := range rows {
			for i := range r {
				r[i] = !r[i]
			}
		}
	}
	return rows
}

func qrWidget(gtx layout.Context, mods [][]bool) layout.Dimensions {
	cols := 0
	for _, r := range mods {
		if len(r) > cols {
			cols = len(r)
		}
	}
	const margin = 2
	// Scale the cell size so the QR code fits in the available width.
	maxW := gtx.Constraints.Max.X
	if maxW <= 0 || maxW > gtx.Dp(280) {
		maxW = gtx.Dp(280)
	}
	cellPx := maxW / (cols + 2*margin)
	if cellPx < 2 {
		cellPx = 2
	}
	if cellPx > 6 {
		cellPx = 6
	}
	cell := cellPx
	w := (cols + 2*margin) * cell
	h := (len(mods) + 2*margin) * cell
	paint.FillShape(gtx.Ops, colWhite, clip.UniformRRect(image.Rect(0, 0, w, h), gtx.Dp(12)).Op(gtx.Ops))
	for y, row := range mods {
		x := 0
		for x < len(row) {
			if !row[x] {
				x++
				continue
			}
			start := x
			for x < len(row) && row[x] {
				x++
			}
			r := image.Rect((margin+start)*cell, (margin+y)*cell, (margin+x)*cell, (margin+y+1)*cell)
			paint.FillShape(gtx.Ops, colQRInk, clip.Rect(r).Op())
		}
	}
	return layout.Dimensions{Size: image.Pt(w, h)}
}

// ---------------------------------------------------------------------------
// GUI entry
// ---------------------------------------------------------------------------

func RunGUI() {
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Baton — Desktop Audio Handoff"),
			app.Size(unit.Dp(1020), unit.Dp(760)),
			app.MinSize(unit.Dp(860), unit.Dp(580)),
		)
		if err := runGioui(w); err != nil {
			fmt.Fprintf(os.Stderr, "gui error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}

func runGioui(w *app.Window) error {
	th := material.NewTheme()
	th.Palette.Bg = colCard
	th.Palette.Fg = colFg
	th.Palette.ContrastBg = colPrimary
	th.Palette.ContrastFg = colWhite

	var ops op.Ops
	c := newCtrls()
	currentTab := 0

	st := &uiState{inv: w.Invalidate}
	st.d = uiData{
		status:    "IDLE",
		statusCol: colMuted,
		owner:     ownerThisPC,
		peer:      "Pixel 8 Pro",
		pairing:   "No active pairing session",
		code:      "--- ---",
		device:    "No paired headphones",
		addr:      "",
	}

	plog := newProtoLog(200)
	plog.Add("// Persistent TCP session ready. Waiting for peer.")

	bt := defaultBluetooth()
	audio := defaultAudio()

	var hc *protocol.HandoffController
	sendFn := func(peerID string, msg map[string]any) error {
		t := protocol.MsgType(msg)
		plog.Add(fmt.Sprintf("[TX → Phone] %s", t))
		if peer := st.getPeer(); peer != nil {
			return peer.Send(msg)
		}
		if t == protocol.TypeHandoffRequest && hc != nil {
			go func() {
				time.Sleep(180 * time.Millisecond)
				plog.Add("[RX ← Phone] HANDOFF_READY (simulated)")
				hc.HandleMessage(peerID, protocol.NewHandoffReady())
				w.Invalidate()
			}()
		}
		return nil
	}
	hc = protocol.NewHandoffController("desktop-1", bt, audio, sendFn)

	go func() {
		for u := range hc.Updates {
			line := fmt.Sprintf("state=%s addr=%s", u.State, u.BtAddress)
			if u.Reason != "" {
				line += " reason=" + string(u.Reason)
			}
			plog.Add(line)
			st.setHandoff(u.State)
		}
	}()

	if devices, _ := bt.PairedDevices(); len(devices) > 0 {
		st.update(func(d *uiData) {
			d.device = devices[0].Name
			d.addr = devices[0].Address
		})
	}

	// Persistent peer listener, started once.
	go func() {
		_, err := protocol.ListenAndServe(":4041", func(conn *protocol.TransportConnection) {
			plog.Add("peer connected on :4041")
			st.setPeer(conn)
			st.update(func(d *uiData) { d.peerOnline = true })
			defer func() {
				st.setPeer(nil)
				conn.Close()
				plog.Add("peer disconnected")
				st.update(func(d *uiData) { d.peerOnline = false })
			}()
			for {
				msg, err := conn.Receive()
				if err != nil {
					return
				}
				plog.Add(fmt.Sprintf("[RX ← Phone] %s", protocol.MsgType(msg)))
				hc.HandleMessage("peer", msg)
				w.Invalidate()
			}
		})
		if err != nil {
			plog.Add("serve error: " + err.Error())
			w.Invalidate()
		}
	}()

	for {
		e := w.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := layout.Context{
				Ops:         &ops,
				Source:      e.Source,
				Metric:      e.Metric,
				Now:         e.Now,
				Constraints: layout.Exact(e.Size),
			}

			for i := range c.navs {
				if c.navs[i].Clicked(gtx) {
					currentTab = i
				}
			}

			s := st.get()

			// ---- Actions ----
			if c.pull.Clicked(gtx) {
				name, addr := s.device, s.addr
				plog.Add(fmt.Sprintf("Pull requested for %s (%s)", name, addr))
				go func() {
					res := hc.Initiate("phone-1", addr, name, protocol.DirPhoneToComputer, true)
					if res.Success {
						plog.Add("handoff SUCCESS")
						st.update(func(d *uiData) { d.owner = ownerThisPC })
						st.setHandoff(protocol.StateActive)
					} else {
						plog.Add(fmt.Sprintf("handoff FAILED: %s %s", res.Reason, res.Detail))
						st.setHandoff(protocol.StateRollback)
					}
				}()
			}

			if c.sendPhone.Clicked(gtx) {
				name, addr, peer := s.device, s.addr, s.peer
				plog.Add("Send to phone requested")
				go func() {
					res := hc.Initiate("phone-1", addr, name, protocol.DirComputerToPhone, true)
					if res.Success {
						st.update(func(d *uiData) { d.owner = peer })
						st.setHandoff(protocol.StateActive)
					} else {
						st.setHandoff(protocol.StateRollback)
					}
				}()
			}

			if c.phoneHandoff.Clicked(gtx) {
				name, addr, peer := s.device, s.addr, s.peer
				plog.Add("[RX ← Phone] HANDOFF_REQUEST (simulated tile)")
				go func() {
					res := hc.Initiate("phone-1", addr, name, protocol.DirComputerToPhone, true)
					if res.Success {
						st.update(func(d *uiData) { d.owner = peer })
					}
				}()
			}

			if c.rollback.Clicked(gtx) {
				plog.Add("[RX] HANDOFF_FAILED reason=TIMEOUT_EXPIRED")
				plog.Add("[TX] ROLLBACK reconnecting original owner")
				st.setHandoff(protocol.StateRollback)
			}

			if c.refresh.Clicked(gtx) {
				if d, err := bt.PairedDevices(); err == nil && len(d) > 0 {
					st.update(func(u *uiData) {
						u.device = d[0].Name
						u.addr = d[0].Address
					})
					plog.Add(fmt.Sprintf("refreshed devices: %d", len(d)))
				}
			}

			if c.clearLog.Clicked(gtx) {
				plog.Clear()
				plog.Add("// Log cleared.")
			}

			if c.genQR.Clicked(gtx) {
				go func() {
					cert, err := protocol.GenerateSelfSignedCert("Baton Desktop")
					if err != nil {
						st.update(func(d *uiData) { d.pairing = "Certificate error: " + err.Error() })
						return
					}
					secret := fmt.Sprintf("baton-%d", time.Now().UnixNano()%100000)
					qr := protocol.QrPayload{
						Address: "127.0.0.1", Port: 4040,
						CertFingerprint: cert.Fingerprint,
						Secret:          secret,
						ExpiresAt:       time.Now().Add(2 * time.Minute),
					}
					enc, _ := qr.Encode()
					qrCode, err := qrcode.New(enc, qrcode.Medium)
					if err != nil {
						plog.Add("qr render error: " + err.Error())
						return
					}
					mods := qrCode.Bitmap()
					code := protocol.ComputeConfirmationCode(cert.Fingerprint)
					if len(code) == 6 {
						code = code[:3] + " " + code[3:]
					}
					st.update(func(d *uiData) {
						d.qr = mods
						d.code = code
						d.pairing = "Waiting for the phone to scan. Expires in 2 minutes."
					})
					plog.Add("pairing QR generated code=" + code)

					tlsCfg, err := protocol.TLSConfigFromPEM(cert.CertPEM, cert.KeyPEM, true)
					if err != nil {
						plog.Add("tls error: " + err.Error())
						return
					}
					ln, err := protocol.ListenAndServeTLS("0.0.0.0:4040", tlsCfg, func(conn *protocol.TransportConnection) {
						defer conn.Close()
						plog.Add("pairing peer connected")
						msg, err := conn.Receive()
						if err != nil {
							return
						}
						if protocol.MsgType(msg) == protocol.TypePairRequest {
							if protocol.GetString(msg, "secret") != secret {
								conn.Send(map[string]any{"type": protocol.TypePairReject, "reason": "invalid secret"})
								return
							}
							_ = conn.Send(map[string]any{
								"type": protocol.TypePairAccept, "deviceName": "Baton Desktop",
								"certFingerprint": cert.Fingerprint, "confirmationCode": code,
							})
							cm, err := conn.Receive()
							if err == nil && protocol.MsgType(cm) == protocol.TypePairConfirm {
								name := protocol.GetString(cm, "deviceName")
								if name == "" {
									name = "Android Device"
								}
								st.update(func(d *uiData) {
									d.pairing = "Paired successfully"
									d.peer = name
								})
								plog.Add("pairing complete")
							}
						}
					})
					if err != nil {
						plog.Add("pairing listener error: " + err.Error())
						return
					}
					st.replaceQRListener(any(ln))
				}()
			}

			if c.revoke.Clicked(gtx) {
				st.replaceQRListener(nil)
				st.update(func(d *uiData) {
					d.pairing = "No active pairing session"
					d.qr = nil
					d.code = "--- ---"
				})
				plog.Add("pairing revoked")
			}

			// ---- Layout ----
			fillRect(gtx, colBg)
			layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return titleBar(gtx, th, s) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return hline(gtx, colBorder) }),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return sidebar(gtx, th, c, currentTab) }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return vline(gtx, colBorder) }),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return content(gtx, th, c, s, plog, currentTab)
						}),
					)
				}),
			)

			e.Frame(&ops)
		}
	}
}

// ---------------------------------------------------------------------------
// Frame sections
// ---------------------------------------------------------------------------

func titleBar(gtx layout.Context, th *material.Theme, s uiData) layout.Dimensions {
	return layout.Inset{Top: 12, Bottom: 12, Left: 20, Right: 20}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return logoMark(gtx, 24) }),
			spx(10),
			layout.Rigid(label(th, "Baton", 15, colFg, font.Bold).Layout),
			spx(8),
			layout.Rigid(label(th, "v0.1", 11, colDim, font.Normal).Layout),
			layout.Flexed(1, empty),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if s.peerOnline {
					return chip(gtx, th, s.peer, colGreenText, colGreenSft, true)
				}
				return chip(gtx, th, "Waiting for phone", colMuted, colCard, true)
			}),
			spx(8),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return chip(gtx, th, "TLS pinned", colIndigoText, colPrimarySft, false)
			}),
		)
	})
}

func sidebar(gtx layout.Context, th *material.Theme, c *ctrls, current int) layout.Dimensions {
	w := gtx.Dp(216)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	names := []string{"Handoff hub", "Pairing and security", "Rules and triggers", "Protocol log"}
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			fillRect(gtx, colBg)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 16, Left: 12, Right: 12, Bottom: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				children := []layout.FlexChild{}
				for i, n := range names {
					i, n := i, n
					children = append(children,
						layout.Rigid(navItem(th, &c.navs[i], n, current == i)),
						sp(4))
				}
				children = append(children,
					layout.Flexed(1, empty),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return surface(gtx, colCard, colBorder, 12, 12, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions { return dot(gtx, colGreen, 8) }),
										spx(8),
										layout.Rigid(label(th, "mDNS active", 12, colGreenText, font.Medium).Layout),
									)
								}),
								sp(4),
								layout.Rigid(monoLabel(th, ":4041 listening", 11, colDim).Layout),
							)
						})
					}),
				)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			})
		}),
	)
}

func navItem(th *material.Theme, btn *widget.Clickable, title string, active bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					switch {
					case active:
						return roundedRect(gtx, colPrimarySft, gtx.Dp(10))
					case btn.Hovered():
						return roundedRect(gtx, colCard, gtx.Dp(10))
					}
					return layout.Dimensions{Size: gtx.Constraints.Min}
				}),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 10, Bottom: 10, Left: 14, Right: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						fg, wt := colMuted, font.Medium
						if active {
							fg, wt = colIndigoText, font.SemiBold
						}
						return label(th, title, 12, fg, wt).Layout(gtx)
					})
				}),
			)
		})
	}
}

func content(gtx layout.Context, th *material.Theme, c *ctrls, s uiData, plog *protoLog, tab int) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			fillRect(gtx, colContent)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 20, Bottom: 8, Left: 24, Right: 24}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return stepperCard(gtx, th, s, c) }),
					sp(16),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						switch tab {
						case 0:
							return scrollPage(gtx, th, &c.pages[0], hubPage(th, s, c))
						case 1:
							return scrollPage(gtx, th, &c.pages[1], pairingPage(th, s, c))
						case 2:
							return scrollPage(gtx, th, &c.pages[2], rulesPage(th, c))
						}
						return protocolPage(gtx, th, c, plog)
					}),
				)
			})
		}),
	)
}

func scrollPage(gtx layout.Context, th *material.Theme, l *widget.List, items []layout.Widget) layout.Dimensions {
	return material.List(th, l).Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
		return layout.Inset{Bottom: 14}.Layout(gtx, items[i])
	})
}

// ---------------------------------------------------------------------------
// Handoff stepper
// ---------------------------------------------------------------------------

func stepperCard(gtx layout.Context, th *material.Theme, s uiData, c *ctrls) layout.Dimensions {
	return surface(gtx, colCard, colBorder, 16, 16, func(gtx layout.Context) layout.Dimensions {
		nodes := make([]layout.FlexChild, 0, len(stateOrder))
		for i, name := range stateOrder {
			i, name := i, name
			nodes = append(nodes, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return stepNode(gtx, th, i, len(stateOrder), s.step, s.failed, stateLabel(name))
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(label(th, "Handoff", 14, colFg, font.Bold).Layout),
					spx(10),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return chip(gtx, th, stateLabel(s.status), s.statusCol, withAlpha(s.statusCol, 40), true)
					}),
					layout.Flexed(1, empty),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return button(gtx, th, &c.phoneHandoff, "Phone handoff", btnSecondary, true, false)
					}),
					spx(8),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return button(gtx, th, &c.rollback, "Timeout rollback", btnDanger, true, false)
					}),
				)
			}),
			sp(18),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{}.Layout(gtx, nodes...)
			}),
		)
	})
}

func stepNode(gtx layout.Context, th *material.Theme, idx, n, cur int, failed bool, name string) layout.Dimensions {
	d := gtx.Dp(16)
	w := gtx.Constraints.Max.X
	lineH := gtx.Dp(2)
	cy := d / 2
	half := w / 2

	seg := func(x0, x1 int, col color.NRGBA) {
		y0 := cy - lineH/2
		paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(x0, y0), Max: image.Pt(x1, y0+lineH)}.Op())
	}
	if idx > 0 {
		col := colBorder
		if idx <= cur {
			col = colPrimary
		}
		seg(0, half, col)
	}
	if idx < n-1 {
		col := colBorder
		if idx < cur {
			col = colPrimary
		}
		seg(half, w, col)
	}

	center := image.Pt(half, cy)
	switch {
	case idx == cur:
		ring := colPrimary
		if failed {
			ring = colRed
		}
		disc(gtx, center, d+gtx.Dp(8), withAlpha(ring, 48))
		disc(gtx, center, d, ring)
		disc(gtx, center, d/2, colWhite)
	case idx < cur:
		disc(gtx, center, d, colPrimary)
	default:
		disc(gtx, center, d, colBorder2)
		disc(gtx, center, d-2*gtx.Dp(2), colCard)
	}

	fg := colDim
	if idx == cur {
		fg = colFg
	} else if idx < cur {
		fg = colMuted
	}
	return layout.Inset{Top: 26}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.N.Layout(gtx, label(th, name, 11, fg, font.Medium).Layout)
	})
}

// ---------------------------------------------------------------------------
// Hub
// ---------------------------------------------------------------------------

func stat(th *material.Theme, title, value string, valueCol color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return surface(gtx, colCard, colBorder, 12, 14, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(label(th, title, 11, colDim, font.Normal).Layout),
				sp(4),
				layout.Rigid(label(th, value, 13, valueCol, font.Medium).Layout),
			)
		})
	}
}

func deviceCard(th *material.Theme, name, sub string, online bool, action layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return surface(gtx, colCard, colBorder, 14, 14, func(gtx layout.Context) layout.Dimensions {
			dotCol, nameCol := colDim, colMuted
			if online {
				dotCol, nameCol = colGreen, colFg
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return dot(gtx, dotCol, 8) }),
				spx(12),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(label(th, name, 13, nameCol, font.Medium).Layout),
						sp(2),
						layout.Rigid(label(th, sub, 11, colDim, font.Normal).Layout),
					)
				}),
				layout.Flexed(1, empty),
				layout.Rigid(action),
			)
		})
	}
}

func hubPage(th *material.Theme, s uiData, c *ctrls) []layout.Widget {
	return []layout.Widget{
		// Active headphones
		func(gtx layout.Context) layout.Dimensions {
			return surface(gtx, colCard, colBorder, 16, 20, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return glyph(gtx, 56) }),
							spx(16),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(label(th, s.device, 20, colFg, font.Bold).Layout),
									sp(6),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return chip(gtx, th, "Multipoint preferred", colGreenText, colGreenSft, false)
											}),
											spx(10),
											layout.Rigid(monoLabel(th, s.addr, 11, colDim).Layout),
										)
									}),
								)
							}),
							layout.Flexed(1, empty),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical, Alignment: layout.End}.Layout(gtx,
									layout.Rigid(label(th, "Playing on", 11, colDim, font.Normal).Layout),
									sp(2),
									layout.Rigid(label(th, s.owner, 14, colFg, font.Medium).Layout),
								)
							}),
						)
					}),
					sp(20),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return button(gtx, th, &c.pull, "Pull headphones to this computer", btnPrimary, false, true)
					}),
					sp(10),
					layout.Rigid(label(th, "Sends a handoff request over local TLS.", 11, colDim, font.Normal).Layout),
				)
			})
		},

		// Link details
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx,
				layout.Flexed(1, stat(th, "Audio", "WASAPI / LDAC", colFg)),
				spx(10),
				layout.Flexed(1, stat(th, "Mode", "Multipoint shortcut", colIndigoText)),
				spx(10),
				layout.Flexed(1, stat(th, "Link", "Connected and ready", colGreenText)),
			)
		},

		// Paired devices
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(label(th, "Paired devices", 14, colFg, font.Bold).Layout),
				layout.Flexed(1, empty),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return button(gtx, th, &c.refresh, "Refresh", btnSecondary, true, false)
				}),
			)
		},
		func(gtx layout.Context) layout.Dimensions {
			phoneSub := "Android, not connected"
			if s.peerOnline {
				phoneSub = "Android, TLS connected"
			}
			return layout.Flex{}.Layout(gtx,
				layout.Flexed(1, deviceCard(th, s.peer, phoneSub, s.peerOnline, func(gtx layout.Context) layout.Dimensions {
					return button(gtx, th, &c.sendPhone, "Send to phone", btnSecondary, true, false)
				})),
				spx(10),
				layout.Flexed(1, deviceCard(th, "Windows Workstation", "Offline on LAN", false, func(gtx layout.Context) layout.Dimensions {
					return label(th, "Unreachable", 11, colDim, font.Normal).Layout(gtx)
				})),
			)
		},
	}
}

// ---------------------------------------------------------------------------
// Pairing
// ---------------------------------------------------------------------------

func pairingPage(th *material.Theme, s uiData, c *ctrls) []layout.Widget {
	statusFg, statusBg := colMuted, colBorder
	switch {
	case strings.HasPrefix(s.pairing, "Paired"):
		statusFg, statusBg = colGreenText, colGreenSft
	case strings.Contains(strings.ToLower(s.pairing), "error"):
		statusFg, statusBg = colRedText, withAlpha(colRed, 34)
	case strings.HasPrefix(s.pairing, "Waiting"):
		statusFg, statusBg = colIndigoText, colPrimarySft
	}

	left := func(gtx layout.Context) layout.Dimensions {
		return surface(gtx, colCard, colBorder, 16, 20, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(label(th, "Pair a new device", 16, colFg, font.Bold).Layout),
				sp(10),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return chip(gtx, th, s.pairing, statusFg, statusBg, true)
				}),
				sp(22),
				layout.Rigid(label(th, "Confirmation code", 12, colMuted, font.Medium).Layout),
				sp(6),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := monoLabel(th, s.code, 34, colIndigoText)
					l.Font.Weight = font.Bold
					return l.Layout(gtx)
				}),
				sp(6),
				layout.Rigid(label(th, "Check that your phone shows the same code before you confirm.", 11, colDim, font.Normal).Layout),
				sp(22),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return button(gtx, th, &c.genQR, "Show pairing QR", btnPrimary, false, true)
				}),
				sp(8),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return button(gtx, th, &c.revoke, "Revoke pairing", btnSecondary, false, true)
				}),
			)
		})
	}

	right := func(gtx layout.Context) layout.Dimensions {
		return surface(gtx, colCard, colBorder, 16, 20, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(label(th, "Scan with Baton on your phone", 13, colFg, font.Medium).Layout),
				sp(14),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if len(s.qr) > 0 {
						return qrWidget(gtx, s.qr)
					}
					sz := gtx.Dp(220)
					gtx.Constraints.Min = image.Pt(sz, sz)
					gtx.Constraints.Max = image.Pt(sz, sz)
					roundedRect(gtx, colBg, gtx.Dp(14))
					return layout.Center.Layout(gtx, label(th, "Your QR code appears here", 12, colDim, font.Normal).Layout)
				}),
				sp(12),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if len(s.qr) == 0 {
						return layout.Dimensions{}
					}
					return label(th, "Expires in 2 minutes", 11, colDim, font.Normal).Layout(gtx)
				}),
			)
		})
	}

	return []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx,
				layout.Flexed(1, left),
				spx(14),
				layout.Flexed(1, right),
			)
		},
	}
}

// ---------------------------------------------------------------------------
// Rules
// ---------------------------------------------------------------------------

func ruleRow(th *material.Theme, title, desc string, sw *widget.Bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(label(th, title, 13, colFg, font.Medium).Layout),
					sp(2),
					layout.Rigid(label(th, desc, 11, colMuted, font.Normal).Layout),
				)
			}),
			spx(16),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				s := material.Switch(th, sw, title)
				s.Color.Enabled = colPrimary
				s.Color.Disabled = colDim
				s.Color.Track = colBorder2
				return s.Layout(gtx)
			}),
		)
	}
}

func rulesPage(th *material.Theme, c *ctrls) []layout.Widget {
	return []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			return surface(gtx, colCard, colBorder, 16, 20, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(label(th, "Automatic rules", 16, colFg, font.Bold).Layout),
					sp(16),
					layout.Rigid(ruleRow(th, "Pull on media playback", "Move the headphones here when media starts playing on this computer.", &c.autoPull)),
					sp(14),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return hline(gtx, colBorder) }),
					sp(14),
					layout.Rigid(ruleRow(th, "Route incoming calls to the phone", "Hand the headphones back to the phone when a call comes in.", &c.callRoute)),
				)
			})
		},
		func(gtx layout.Context) layout.Dimensions {
			return surface(gtx, colCard, colBorder, 16, 20, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(label(th, "Releasing the headphones on Android", 14, colFg, font.Bold).Layout),
					sp(4),
					layout.Rigid(label(th, "Baton tries each method in order until one works.", 11, colMuted, font.Normal).Layout),
					sp(14),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return chip(gtx, th, "Multipoint", colIndigoText, colPrimarySft, false)
							}),
							spx(10),
							layout.Rigid(label(th, "then", 11, colDim, font.Normal).Layout),
							spx(10),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return chip(gtx, th, "Shizuku", colMuted, colBorder, false)
							}),
							spx(10),
							layout.Rigid(label(th, "then", 11, colDim, font.Normal).Layout),
							spx(10),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return chip(gtx, th, "Pause playback", colMuted, colBorder, false)
							}),
						)
					}),
				)
			})
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx,
				layout.Flexed(1, stat(th, "Cooldown between handoffs", "30 seconds", colFg)),
				spx(10),
				layout.Flexed(1, stat(th, "Rollback timeout", "10 seconds", colFg)),
			)
		},
	}
}

// ---------------------------------------------------------------------------
// Protocol log
// ---------------------------------------------------------------------------

func logColor(line string) color.NRGBA {
	l := strings.ToLower(line)
	switch {
	case strings.Contains(l, "failed") || strings.Contains(l, "error") || strings.Contains(l, "rollback"):
		return colRedText
	case strings.Contains(line, "[TX"):
		return colIndigoText
	case strings.Contains(line, "[RX"):
		return colGreenText
	case strings.Contains(line, "//"):
		return colDim
	}
	return colMuted
}

func protocolPage(gtx layout.Context, th *material.Theme, c *ctrls, plog *protoLog) layout.Dimensions {
	lines := plog.Lines()
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return chip(gtx, th, "Length-prefixed JSON over pinned TLS", colIndigoText, colPrimarySft, true)
				}),
				layout.Flexed(1, empty),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return button(gtx, th, &c.clearLog, "Clear log", btnSecondary, true, false)
				}),
			)
		}),
		sp(12),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = gtx.Constraints.Max
			return surface(gtx, colBg, colBorder, 14, 14, func(gtx layout.Context) layout.Dimensions {
				return material.List(th, &c.logList).Layout(gtx, len(lines), func(gtx layout.Context, i int) layout.Dimensions {
					return layout.Inset{Bottom: 3}.Layout(gtx, monoLabel(th, lines[i], 12, logColor(lines[i])).Layout)
				})
			})
		}),
	)
}