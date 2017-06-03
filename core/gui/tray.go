// +build !linux

// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gui

import (
	"sync/atomic"

	"github.com/getlantern/systray"

	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/core/gui/icon"
)

var trayRunning int32 // != 0 means a system tray is running

type TrayItem struct {
	*systray.MenuItem
	title       string
	checked     bool
	disabled    bool
	clickEvents []TrayItemClickEvent
}

type TrayItemClickEvent func(*TrayItem)

func newTrayItem(title string) *TrayItem {
	// MenuItem's fields are not exported,
	// I could modify the source code to completes my needs
	// but I will not because the sys tram is only one
	// and its items are not shown until .Show()
	// So I will use the AddMenuItem because it adds them with a specific order
	// and I want to control the order, so this TrayItem
	// will be a wrapper of the *systray.MenuItem and it will be shown as independed when tray host.Show called.
	return &TrayItem{title: title}
}

func (i *TrayItem) Check() {
	i.checked = true
	if i.MenuItem != nil {
		i.MenuItem.Check()
	}
}

func (i *TrayItem) Uncheck() {
	i.checked = false
	if i.MenuItem != nil {
		i.MenuItem.Uncheck()
	}
}

func (i *TrayItem) Checked() bool {
	return i.checked
}

func (i *TrayItem) Disable() {
	i.disabled = true
	if i.MenuItem != nil {
		i.MenuItem.Disable()
	}
}

func (i *TrayItem) Enable() {
	i.disabled = false
	if i.MenuItem != nil {
		i.MenuItem.Enable()
	}
}

func (i *TrayItem) Disabled() bool {
	return i.disabled
}

func (i *TrayItem) SetTitle(title string) *TrayItem {
	i.title = title
	if i.MenuItem != nil {
		i.MenuItem.SetTitle(title)
		i.MenuItem.SetTooltip(title)
	}
	return i
}

func (i *TrayItem) SetToolTip(tooltip string) *TrayItem {
	return i.SetTitle(tooltip)
}

// always checked by-default
func newTrayItemCheckBox(checkedTitle string, onCheck TrayItemClickEvent,
	unCheckedTitle string, onUnCheck TrayItemClickEvent) *TrayItem {

	item := newTrayItem(checkedTitle)
	item.Check()

	item.OnClick(func(i *TrayItem) {
		if item.Checked() {
			item.SetTitle(unCheckedTitle)
			item.Uncheck()
			onUnCheck(i)
		} else {
			item.SetTitle(checkedTitle)
			item.Check()
			onCheck(i)
		}
	})

	return item
}

const (
	sepLine = "────────────────"
)

func newTrayItemSeparator() *TrayItem {
	item := newTrayItem(sepLine)
	item.Disable()
	return item
}

func (i *TrayItem) show() {
	activeItem := systray.AddMenuItem(i.title, i.title)
	if i.disabled {
		activeItem.Disable()
	}

	if i.checked {
		activeItem.Check()
	}

	i.MenuItem = activeItem
	go func() {
		for {
			<-activeItem.ClickedCh
			i.fireClick()
		}
	}()
}

func (i *TrayItem) OnClick(callback TrayItemClickEvent) *TrayItem {
	i.clickEvents = append(i.clickEvents, callback)
	return i
}

func (i *TrayItem) fireClick() {
	for _, cb := range i.clickEvents {
		cb(i)
	}
}

type TrayHost struct {
	items            []*TrayItem // useless
	version          string
	shutdownServerCb func()
	startServerCb    func()
	hideCb           func()
}

var Tray = defaultTrayHost()

func defaultTrayHost() *TrayHost {
	t := new(TrayHost)
	return t
}

func (t *TrayHost) putItem(item *TrayItem) {
	t.items = append(t.items, item)
}

func (t *TrayHost) PutItem(title string) *TrayItem {
	item := newTrayItem(title)
	t.putItem(item)
	return item
}

func (t *TrayHost) PutCheckBox(checkedTitle string, onCheck TrayItemClickEvent,
	unCheckedTitle string, onUnCheck TrayItemClickEvent) *TrayItem {
	item := newTrayItemCheckBox(checkedTitle, onCheck, unCheckedTitle, onUnCheck)
	t.putItem(item)
	return item
}

func (t *TrayHost) PutSeparator() *TrayItem {
	item := newTrayItemSeparator()
	t.putItem(item)
	return item
}

func (t *TrayHost) SetVersion(v string) *TrayHost {
	t.version = v
	return t
}

func (t *TrayHost) OnServerStatusChange(start func(), shutdown func()) *TrayHost {
	t.startServerCb = start
	t.shutdownServerCb = shutdown
	return t
}

func (t *TrayHost) OnHide(cb func()) *TrayHost {
	t.hideCb = cb
	return t
}

func (t *TrayHost) Hide() {
	atomic.StoreInt32(&trayRunning, -1)
	systray.Quit()
	if t.hideCb != nil {
		t.hideCb()
	}
}

func (t *TrayHost) Show() error {
	if running := atomic.LoadInt32(&trayRunning); running != 0 {
		return errors.New("A system tray is already running, please close that first")
	}

	topItems := make([]*TrayItem, 0) // dynamic because we don't know if status btn will be shown

	versionBtn := newTrayItem("Version " + t.version)
	versionBtn.Disable()

	topItems = append(topItems, versionBtn)
	// if server status listeners have been registered, then show the online/offline button
	// otherwise not.
	if t.startServerCb != nil && t.shutdownServerCb != nil {
		statusBtn := newTrayItemCheckBox("Online", t.startServerClicked, "Offline", t.shutdownServerClicked)
		topItems = append(topItems, statusBtn)
	}
	topItems = append(topItems, newTrayItemSeparator())

	t.items = append(topItems, t.items...)
	t.PutItem("Hide").OnClick(t.hideClicked)

	systray.Run(t.onTrayReady)
	return nil
}

func (t *TrayHost) onTrayReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("Iris web server")
	systray.SetTooltip("Iris")

	for _, item := range t.items {
		item.show()
	}

	atomic.StoreInt32(&trayRunning, 1)
}

func (t *TrayHost) DisableItem(index int) {
	if len(t.items)-1 > index {
		t.items[index].Disable()
	}
}

func (t *TrayHost) startServerClicked(item *TrayItem) {
	if t.startServerCb != nil {
		t.startServerCb()
	}
}

func (t *TrayHost) shutdownServerClicked(item *TrayItem) {
	if t.shutdownServerCb != nil {
		t.shutdownServerCb()
	}
}

func (t *TrayHost) hideClicked(item *TrayItem) {
	t.Hide()
}
