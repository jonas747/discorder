package discorder

import (
	"github.com/jonas747/discorder/common"
	"github.com/jonas747/discorder/ui"
	"strconv"
	"strings"
)

type CommandExecWindow struct {
	*ui.BaseEntity
	app        *App
	menuWindow *ui.MenuWindow
	command    Command
}

func NewCommandExecWindow(layer int, app *App, command Command) *CommandExecWindow {
	execWindow := &CommandExecWindow{
		BaseEntity: &ui.BaseEntity{},
		app:        app,
		menuWindow: ui.NewMenuWindow(layer, app.ViewManager.UIManager, false),
		command:    command,
	}

	execWindow.menuWindow.Transform.AnchorMax = common.NewVector2F(1, 1)
	execWindow.menuWindow.Transform.Top = 1
	execWindow.menuWindow.Transform.Bottom = 2

	execWindow.menuWindow.Window.Title = "Execute command"
	execWindow.menuWindow.Window.Footer = ":)"

	app.ApplyThemeToMenu(execWindow.menuWindow)

	execWindow.Transform.AddChildren(execWindow.menuWindow)

	execWindow.Transform.AnchorMin = common.NewVector2F(0.1, 0)
	execWindow.Transform.AnchorMax = common.NewVector2F(0.9, 1)

	app.ViewManager.UIManager.AddWindow(execWindow)

	execWindow.GenMenu()

	return execWindow
}

func (cew *CommandExecWindow) Destroy() {
	cew.app.ViewManager.UIManager.RemoveWindow(cew)
	cew.DestroyChildren()
}

func (cew *CommandExecWindow) GenMenu() {
	items := make([]*ui.MenuItem, 0)
	for _, arg := range cew.command.GetArgs() {
		helper := &ui.MenuItem{
			Name: arg.Name,
			Info: arg.Description,
		}
		input := &ui.MenuItem{
			Name:      arg.Name,
			Info:      arg.Description,
			IsInput:   true,
			InputType: arg.Datatype,
			UserData:  arg,
		}
		items = append(items, helper, input)
	}

	exec := &ui.MenuItem{
		Name: "Execute",
		Info: "Execute the commadn with specified args",
	}
	items = append(items, exec)
	cew.menuWindow.SetOptions(items)
}

func (cew *CommandExecWindow) Select() {
	element := cew.menuWindow.GetHighlighted()
	if element == nil {
		return
	}

	if element.IsCategory {
		cew.menuWindow.Select()
		return
	}

	if element.Name == "Execute" {
		cew.Execute()
	}
}

func (cew *CommandExecWindow) Execute() {
	args := make(map[string]interface{})
	for _, item := range cew.menuWindow.Options {
		if !item.IsInput {
			continue
		}
		buf := item.Input.TextBuffer
		switch item.InputType {
		case ui.DataTypeString:
			args[item.Name] = buf
		case ui.DataTypeBool:
			lowerBuf := strings.ToLower(buf)
			b, _ := strconv.ParseBool(lowerBuf)
			args[item.Name] = b
		case ui.DataTypeInt:
			i, _ := strconv.ParseInt(buf, 10, 64)
			args[item.Name] = i
		case ui.DataTypeFloat:
			f, _ := strconv.ParseFloat(buf, 64)
			args[item.Name] = f
		}
	}

	cew.app.RunCommand(cew.command, Arguments(args))
	cew.Transform.Parent.RemoveChild(cew, true)
}
