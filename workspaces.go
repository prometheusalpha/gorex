package main

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Workspace is a set of tabs, as those of a project: the sidebar lists
// them, and the window shows the tabs of one.
type Workspace struct {
	ID   int
	Name string // the name the user gave it, "" to follow its focused pane
	// Dir is where its new panes open when no pane has the focus.
	Dir    string
	Tabs   []*Tab
	Active int
}

const sidebarW = 196

func (w *Workspace) tab() *Tab {
	if w.Active >= 0 && w.Active < len(w.Tabs) {
		return w.Tabs[w.Active]
	}
	return nil
}

// label names a workspace by the folder its focused pane is in, as that
// pane cds, rather than the one it opened in.
func (w *Workspace) label() string {
	if w.Name != "" {
		return w.Name
	}
	if t := w.tab(); t != nil && t.Focus != nil && t.Focus.info.Dir != "" {
		return filepath.Base(t.Focus.info.Dir)
	}
	if w.Dir != "" {
		return filepath.Base(w.Dir)
	}
	return "Default"
}

// ws returns the workspace that shows. A window starts with none until it
// opens a tab, which makes one.
func (a *App) ws() *Workspace {
	if a.cur == nil {
		a.cur = &Workspace{ID: a.id()}
		a.workspaces = append(a.workspaces, a.cur)
	}
	return a.cur
}

// allTabs returns the tabs of every workspace.
func (a *App) allTabs() []*Tab {
	var ts []*Tab
	for _, w := range a.workspaces {
		ts = append(ts, w.Tabs...)
	}
	return ts
}

func (a *App) workspaceOf(t *Tab) *Workspace {
	for _, w := range a.workspaces {
		if slices.Contains(w.Tabs, t) {
			return w
		}
	}
	return nil
}

// newWorkspace opens a workspace of dir, with a shell there, after the one
// that shows. New Workspace takes the directory of the one that shows,
// rather than asking for one.
func (a *App) newWorkspace(dir string) *Workspace {
	w := &Workspace{ID: a.id(), Dir: dir}
	at := slices.Index(a.workspaces, a.cur) + 1
	a.workspaces = slices.Insert(a.workspaces, at, w)
	a.cur = w
	a.renaming = nil
	a.newTab(dir)
	return w
}

func (a *App) selectWorkspace(i int) {
	if i < 0 || i >= len(a.workspaces) {
		return
	}
	a.cur = a.workspaces[i]
	a.renaming = nil
	if t := a.tab(); t != nil {
		if t.Focus != nil {
			t.Focus.attention = false
		}
		a.focusReq = t.Focus
	}
	a.changed()
}

func (a *App) cycleWorkspace(d int) {
	if n := len(a.workspaces); n > 1 {
		i := slices.Index(a.workspaces, a.cur)
		a.selectWorkspace(((i+d)%n + n) % n)
	}
}

// showWorkspace shows the nth workspace, from 1; 9 is the last.
func (a *App) showWorkspace(n int) {
	if n == 9 {
		a.selectWorkspace(len(a.workspaces) - 1)
	} else {
		a.selectWorkspace(n - 1)
	}
}

// removeWorkspace takes out a workspace whose tabs are gone; the one before
// it shows in its place.
func (a *App) removeWorkspace(w *Workspace) {
	i := slices.Index(a.workspaces, w)
	if i < 0 {
		return
	}
	a.workspaces = slices.Delete(a.workspaces, i, i+1)
	if a.renamingWS == w {
		a.renamingWS = nil
	}
	if a.cur == w {
		a.cur = nil
		if len(a.workspaces) > 0 {
			a.selectWorkspace(max(i-1, 0))
		}
	}
	a.changed()
}

// closeWorkspace ends the sessions of a workspace's panes; the last
// workspace stays, as the window would close with it.
func (a *App) closeWorkspace(w *Workspace) {
	if len(a.workspaces) < 2 {
		return
	}
	for _, t := range slices.Clone(w.Tabs) {
		a.closeTab(t)
	}
}

func (a *App) startRenameWorkspace(w *Workspace) {
	a.renaming = nil
	a.renamingWS, a.renameText, a.renameFocus = w, w.Name, false
	a.focusReq = nil // the field takes the focus
}

// sidebar lists the workspaces, and under them the agents of every
// workspace.
func (a *App) sidebar(c *ui.Context, k *colors) {
	ui.Scroll(c).Width(sidebarW).Shrink(0).MinHeight(0).Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			sidebarHeading(c, k, "Workspaces", func() {
				if iconButton(c, k, "plus", "New Workspace", 24, 15).Tooltip(tip("New Workspace", &cmdNewWorkspace)).Clicked() {
					a.newWorkspace(a.currentDir())
				}
			})
			ui.Column(c).Gap(2).Role(ui.RoleList).Label("Workspaces").Children(func() {
				for i, w := range slices.Clone(a.workspaces) {
					a.workspaceItem(c, k, i, w)
				}
			})
			ui.Box(c).Height(10)
			sidebarHeading(c, k, "Agents", nil)
			a.agentList(c, k)
		})
	})
}

// sidebarHeading is the heading of a list of the sidebar, with buttons
// on its right.
func sidebarHeading(c *ui.Context, k *colors, title string, buttons func()) {
	ui.Row(c).Height(30).Padding(0, 2, 0, 10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(11.5).FontWeight(600).TextColor(k.textFaint).Grow(1)
		if buttons != nil {
			buttons()
		}
	})
}

// workspaceItem draws a workspace's row of the sidebar.
func (a *App) workspaceItem(c *ui.Context, k *colors, i int, w *Workspace) {
	active := w == a.cur
	e := ui.Row(c.Key(w.ID)).Padding(7, 8, 7, 10).Gap(8).AlignItems(ui.Center).Radius(10).
		Cursor(ui.CursorPointer).Role(ui.RoleListItem).Selected(active).Label(w.label())
	if active {
		e.Background(k.tabActive).Shadow(0, 1, 2, 0, k.shadow).Shadow(0, 2, 8, 0, k.shadow)
	} else if e.Hovered() {
		e.Background(k.hover)
	}
	e.Transition(ui.ElementTransition{Colors: true, Duration: 160 * time.Millisecond})
	if e.Clicked() && !active {
		a.selectWorkspace(i)
	}
	if e.DoubleClicked() {
		a.selectWorkspace(i)
		a.startRenameWorkspace(w)
	}
	e.ContextMenu(func(m *ui.Menu) { a.workspaceMenu(m, w) })
	e.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			if a.renamingWS == w {
				a.renameWorkspaceField(c, w)
			} else {
				col := k.text
				if !active {
					col = k.textMuted
				}
				ui.Text(c, w.label()).FontSize(13).FontWeight(600).TextColor(col).SingleLine().Ellipsis("…")
			}
			if g := a.gitOf(w.dir()); g.Branch != "" {
				gitLine(c, k, g)
			}
		})
	})
}

// gitLine shows a workspace's branch, and the commits it is ahead of its
// upstream and behind it.
func gitLine(c *ui.Context, k *colors, g gitState) {
	ui.Row(c).Gap(4).MinWidth(0).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icon("git-branch")).Size(11, 11).TextColor(k.textFaint).Shrink(0)
		ui.Text(c, g.Branch).FontSize(11.5).TextColor(k.textFaint).SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
		if g.Ahead > 0 {
			ui.Text(c, strconv.Itoa(g.Ahead)+"↑").FontSize(11.5).FontWeight(600).TextColor(k.busy).SingleLine().Shrink(0)
		}
		if g.Behind > 0 {
			ui.Text(c, strconv.Itoa(g.Behind)+"↓").FontSize(11.5).FontWeight(600).TextColor(k.attention).SingleLine().Shrink(0)
		}
	})
}

// renameWorkspaceField edits the name of a workspace in place, as
// renameField a tab's.
func (a *App) renameWorkspaceField(c *ui.Context, w *Workspace) {
	in := ui.TextInput(c, &a.renameText).Placeholder(w.label()).MinWidth(0).Height(22).FontSize(13).AutoFocus()
	done := func(keep bool) {
		if keep {
			w.Name = strings.TrimSpace(a.renameText)
			a.changed()
		}
		a.renamingWS, a.renameFocus = nil, false
		if t := a.tab(); t != nil {
			a.focusReq = t.Focus
		}
	}
	switch {
	case in.Shortcut(0, ui.KeyEscape):
		done(false)
	case in.Submitted():
		done(true)
	case in.Focused():
		a.renameFocus = true
	case a.renameFocus:
		done(true) // the focus went elsewhere
	}
}

func (a *App) workspaceMenu(m *ui.Menu, w *Workspace) {
	if m.Item("Rename Workspace…").Chosen() {
		a.selectWorkspace(slices.Index(a.workspaces, w))
		a.startRenameWorkspace(w)
	}
	if w.Name != "" && m.Item("Reset Name").Chosen() {
		w.Name = ""
		a.changed()
	}
	m.Separator()
	if m.Item("New Workspace").Chosen() {
		a.newWorkspace(w.dir())
	}
	if m.Item("Close Workspace").Disabled(len(a.workspaces) < 2).Chosen() {
		a.laterService(func() { a.closeWorkspace(w) })
	}
}
