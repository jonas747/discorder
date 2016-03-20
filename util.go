package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/nsf/termbox-go"
	"log"
	"time"
	"unicode/utf8"
)

type ListSelection struct {
	app          *App
	Options      []string
	Header       string
	curSelection int
	marked       []int
}

func (s *ListSelection) HandleInput(event termbox.Event) {
	if event.Type == termbox.EventKey {
		if event.Key == termbox.KeyArrowUp {
			s.curSelection--
			if s.curSelection < 0 {
				s.curSelection = 0
			}
		} else if event.Key == termbox.KeyArrowDown {
			s.curSelection++
			if s.curSelection >= len(s.Options) {
				s.curSelection = len(s.Options) - 1
			}
		} else if event.Key == termbox.KeyBackspace || event.Key == termbox.KeyBackspace2 {
			s.app.currentState = &StateNormal{s.app}
		}
	}
}

func (s *ListSelection) RefreshDisplay() {
	if s.Header == "" {
		s.Header = "Select an item"
	}
	if s.marked == nil {
		s.marked = []int{}
	}
	s.app.CreateListWindow(s.Header, s.Options, s.curSelection, s.marked)
}

func (s *ListSelection) GetCurrentSelection() string {
	return s.Options[s.curSelection]
}

// For logs
func (app *App) Write(p []byte) (n int, err error) {
	cop := string(p)

	// since we might log from the same goroutine deadlocks may occour, should probably do a queue system or something instead...
	go func() {
		app.logChan <- cop
	}()

	if app.logFile != nil {
		app.logFileLock.Lock()
		defer app.logFileLock.Unlock()
		app.logFile.Write(p)
	}

	return len(p), nil
}

func (app *App) HandleTextInput(event termbox.Event) {
	if event.Type == termbox.EventKey {

		switch event.Key {

		case termbox.KeyArrowLeft:
			app.currentCursorLocation--
			if app.currentCursorLocation < 0 {
				app.currentCursorLocation = 0
			}
		case termbox.KeyArrowRight:
			app.currentCursorLocation++
			bufLen := utf8.RuneCountInString(app.currentTextBuffer)
			if app.currentCursorLocation > bufLen {
				app.currentCursorLocation = bufLen
			}
		case termbox.KeyBackspace, termbox.KeyBackspace2:
			bufLen := utf8.RuneCountInString(app.currentTextBuffer)
			if bufLen == 0 {
				return
			}
			if app.currentCursorLocation == bufLen {
				_, size := utf8.DecodeLastRuneInString(app.currentTextBuffer)
				app.currentCursorLocation--
				app.currentTextBuffer = app.currentTextBuffer[:len(app.currentTextBuffer)-size]
			} else if app.currentCursorLocation == 1 {
				_, size := utf8.DecodeRuneInString(app.currentTextBuffer)
				app.currentCursorLocation--
				app.currentTextBuffer = app.currentTextBuffer[size:]
			} else if app.currentCursorLocation == 0 {
				return
			} else {
				runeSlice := []rune(app.currentTextBuffer)
				newSlice := append(runeSlice[:app.currentCursorLocation], runeSlice[app.currentCursorLocation+1:]...)
				app.currentTextBuffer = string(newSlice)
				app.currentCursorLocation--
			}
		default:
			char := event.Ch
			if event.Key == termbox.KeySpace {
				char = ' '
			} else if event.Key == termbox.Key(0) && event.Mod == termbox.ModAlt && char == 0 {
				char = '@' // Just temporary workaround for non american keyboards on windows
				// So they're atleast able to log in
			}

			bufLen := utf8.RuneCountInString(app.currentTextBuffer)
			if app.currentCursorLocation == bufLen {
				app.currentTextBuffer += string(char)
				app.currentCursorLocation++
			} else if app.currentCursorLocation == 0 {
				app.currentTextBuffer = string(char) + app.currentTextBuffer
				app.currentCursorLocation++
			} else {
				bufSlice := []rune(app.currentTextBuffer)
				bufCopy := ""

				for i := 0; i < len(bufSlice); i++ {
					if i == app.currentCursorLocation {
						bufCopy += string(char)
					}
					bufCopy += string(bufSlice[i])
				}
				app.currentTextBuffer = bufCopy
				app.currentCursorLocation++
			}
		}

	}
}

func (app *App) GetHistory(channelId string, limit int, beforeId, afterId string) {
	state := app.session.State
	channel, err := state.Channel(channelId)
	if err != nil {
		log.Println("History error: ", err)
		return
	}

	// func (s *Session) ChannelMessages(channelID string, limit int, beforeID, afterID string) (st []*Message, err error)
	resp, err := app.session.ChannelMessages(channelId, limit, beforeId, afterId)
	if err != nil {
		log.Println("History error: ", err)
		return
	}

	state.Lock()
	defer state.Unlock()

	newMessages := make([]*discordgo.Message, 0)

	if len(channel.Messages) < 1 && len(resp) > 0 {
		for i := len(resp) - 1; i >= 0; i-- {
			channel.Messages = append(channel.Messages, resp[i])
		}
		return
	} else if len(resp) < 1 {
		return
	}

	nextNewMessageIndex := len(resp) - 1
	nextOldMessageIndex := 0

	for {
		newOut := false
		oldOut := false
		var nextOldMessage *discordgo.Message
		if nextOldMessageIndex >= len(channel.Messages) {
			oldOut = true
		} else {
			nextOldMessage = channel.Messages[nextOldMessageIndex]
		}

		var nextNewMessage *discordgo.Message
		if nextNewMessageIndex < 0 {
			newOut = true
		} else {
			nextNewMessage = resp[nextNewMessageIndex]
		}

		if newOut && !oldOut {
			newMessages = append(newMessages, nextOldMessage)
			nextOldMessageIndex++
			continue
		} else if !newOut && oldOut {
			newMessages = append(newMessages, nextNewMessage)
			nextNewMessageIndex--
			continue
		} else if newOut && oldOut {
			break
		}

		if nextNewMessage.ID == nextOldMessage.ID {
			newMessages = append(newMessages, nextNewMessage)
			nextNewMessageIndex--
			nextOldMessageIndex++
			continue
		}

		parsedNew, _ := time.Parse(DiscordTimeFormat, nextNewMessage.Timestamp)
		parsedOld, _ := time.Parse(DiscordTimeFormat, nextOldMessage.Timestamp)

		if parsedNew.Before(parsedOld) {
			newMessages = append(newMessages, nextNewMessage)
			nextNewMessageIndex--
		} else {
			newMessages = append(newMessages, nextOldMessage)
			nextOldMessageIndex++
		}
	}
	channel.Messages = newMessages
	log.Println("History processing completed!")
}

// Returns true if removed, otherwise it didnt exist
func (app *App) RemoveListeningChannel(chId string) bool {
	index := -1
	for i, listening := range app.listeningChannels {
		if listening == chId {
			index = i
			break
		}
	}

	if index != -1 {
		// Remove
		if index == 0 {
			app.listeningChannels = app.listeningChannels[1:]
		} else if index == len(app.listeningChannels)-1 {
			app.listeningChannels = app.listeningChannels[:len(app.listeningChannels)-1]
		} else {
			app.listeningChannels = append(app.listeningChannels[:index], app.listeningChannels[index+1:]...)
		}
		return true
	}
	return false
}

// Returns true if added, false if allready added before
func (app *App) AddListeningChannel(chId string) bool {
	for _, v := range app.listeningChannels {
		if v == chId {
			return false
		}
	}
	app.listeningChannels = append(app.listeningChannels, chId)
	go app.GetHistory(chId, 50, "", "")

	return true
}

func (app *App) ToggleListeningChannel(chId string) {
	if !app.AddListeningChannel(chId) {
		app.RemoveListeningChannel(chId)
	}
}