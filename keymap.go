package main

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	// standard movement between app pages

	Quit  key.Binding
	Enter key.Binding
	Back  key.Binding
	Help  key.Binding

	// moving inside a page?
	// Up   key.Binding
	// Down key.Binding
}

var DefaultKeyMap = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "quit"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "enter"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}
