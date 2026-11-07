package main

import (
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// agentState is what an agent does, as the sidebar shows it.
type agentState int

const (
	agentIdle      agentState = iota
	agentWorking              // printing
	agentAttention            // rang the bell, or finished out of sight
)

func (s agentState) String() string {
	return [...]string{"Idle", "Working", "Needs attention"}[s]
}

// agent is a pane that runs a coding agent, and where it is.
type agent struct {
	pane *Pane
	tab  *Tab
	ws   *Workspace
}

// agents returns the agents of every workspace, in the order of the
// workspaces, their tabs and panes: rows that moved as agents change
// state would move under the pointer.
func (a *App) agents() []agent {
	var out []agent
	for _, w := range a.workspaces {
		for _, t := range w.Tabs {
			for _, p := range t.panes() {
				if !p.closed && !p.info.Idle && programOf(p.info.Program).Agent {
					out = append(out, agent{p, t, w})
				}
			}
		}
	}
	return out
}

func (a *App) agentStateOf(c *ui.Context, p *Pane) agentState {
	switch a.statusOf(c, p) {
	case statusAttention:
		return agentAttention
	case statusRunning:
		return agentWorking
	}
	return agentIdle
}

// goTo shows a pane, in whichever workspace and tab it is, and gives it
// the focus.
func (a *App) goTo(p *Pane) {
	w := a.workspaceOf(p.Tab)
	if w == nil {
		return
	}
	if w != a.cur {
		a.selectWorkspace(slices.Index(a.workspaces, w))
	}
	a.selectTab(slices.Index(w.Tabs, p.Tab))
	p.Tab.setFocus(p)
	a.focusReq = p
}

// agentList lists the agents of every workspace: what each does, and
// where; clicking one goes to its pane.
func (a *App) agentList(c *ui.Context, k *colors) {
	agents := a.agents()
	ui.Column(c).Gap(2).Role(ui.RoleList).Label("Agents").Children(func() {
		if len(agents) == 0 {
			ui.Text(c, "No agents running").FontSize(12).TextColor(k.textFaint).Padding(2, 10)
		}
		for _, ag := range agents {
			a.agentItem(c, k, ag)
		}
	})
}

func (a *App) agentItem(c *ui.Context, k *colors, ag agent) {
	p := ag.pane
	name, _ := p.label()
	state := a.agentStateOf(c, p)
	shown := ag.ws == a.cur && ag.tab == a.tab() && ag.tab.Focus == p
	e := ui.Row(c.Key(p.ID)).Padding(6, 8, 6, 8).Gap(9).AlignItems(ui.Center).Radius(10).
		Cursor(ui.CursorPointer).Role(ui.RoleListItem).Selected(shown).Label(name)
	if shown {
		e.Background(k.tabActive).Shadow(0, 1, 2, 0, k.shadow)
	} else if e.Hovered() {
		e.Background(k.hover)
	}
	e.Transition(ui.ElementTransition{Colors: true, Duration: 160 * time.Millisecond})
	if e.Clicked() {
		a.goTo(p)
	}
	stateColor := k.textFaint
	switch state {
	case agentWorking:
		stateColor = k.busy
	case agentAttention:
		stateColor = k.attention
	}
	e.Children(func() {
		prog := paneProgram(p)
		ui.Box(c).Size(tileW+2, tileH+2).Shrink(0).Draw(func(pt *ui.Painter, r ui.Rect) {
			drawTile(pt, k, ui.Rect{X: r.X + 1, Y: r.Y + 1, W: tileW, H: tileH}, prog, true)
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			col := k.text
			if !shown {
				col = k.textMuted
			}
			ui.Text(c, name).FontSize(12.5).FontWeight(600).TextColor(col).SingleLine().Ellipsis("…")
			ui.Row(c).Gap(4).MinWidth(0).Children(func() {
				ui.Text(c, state.String()).FontSize(11.5).FontWeight(500).TextColor(stateColor).SingleLine().Shrink(0)
				ui.Text(c, "· "+ag.ws.label()).FontSize(11.5).TextColor(k.textFaint).SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
			})
		})
		switch state {
		case agentWorking:
			activityDot(c, k.busy, 6)
		case agentAttention:
			ui.Box(c).Size(7, 7).Radius(4).Background(k.attention).Margin(0, 2, 0, 0)
		}
	})
}
