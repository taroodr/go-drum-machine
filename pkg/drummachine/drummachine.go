package drummachine

import (
	"context"
	"fmt"

	ui "github.com/gizak/termui/v3"
	"github.com/taroodr/fluidsynth2"

	"github.com/pkg/errors"

	"github.com/mattn/go-tty"
	"github.com/taroodr/my-drum-machine/pkg/kits"
	"github.com/taroodr/my-drum-machine/pkg/midi"
)

type Synth struct {
	synth  fluidsynth2.Synth
	driver fluidsynth2.AudioDriver
	Kit
	// Sequencer *fluidsynth2
}

type Kit interface {
	GetSoundPath() (string, error)

	Render() error
	Close() error
	PollEvents() <-chan ui.Event

	Kick() *midi.NoteMessage
	Snare() *midi.NoteMessage
	Clap() *midi.NoteMessage
	HighHatOpen() *midi.NoteMessage
	HighHatClosed() *midi.NoteMessage
	TomLow() *midi.NoteMessage
	TomHigh() *midi.NoteMessage
}

func GetKit(name string) (Kit, error) {
	switch name {
	case kits.KitName:
		return &kits.EightOhEight{}, nil
	// case nineohnine.KitName:
	// 	return &nineohnine.NineOhNine{}, nil

	default:
		return &kits.EightOhEight{}, nil
	}
}

func NewDrumMachine(kitName string) (*Synth, error) {
	setting := fluidsynth2.NewSettings()
	synth := fluidsynth2.NewSynth(setting)

	driver := fluidsynth2.NewAudioDriver(setting, synth)

	seq := fluidsynth2.NewSequencer()
	seq.RegisterSynth(synth)

	kit, err := GetKit(kitName)
	if err != nil {
		return nil, err
	}

	soundPath, err := kit.GetSoundPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get soundpath")
	}

	if _, err := synth.SFLoad(soundPath, true); err != nil {
		return nil, fmt.Errorf("failed to load soundfile")
	}

	return &Synth{
		synth:  synth,
		driver: driver,
		Kit:    kit,
	}, nil
}

func (s *Synth) SetupInstrument(ctx context.Context) error {
	tty, err := tty.Open()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to open tty"))
	}
	defer tty.Close()

	fmt.Println("Press key to play")

	if err := s.Kit.Render(); err != nil {
		return err
	}
	defer s.Kit.Close()

	endCh := make(chan error)
	defer close(endCh)

	// uiEvents := s.Kit.PollEvents()

	// if err := s.Kit.Render(); err != nil {
	// 	return errors.Wrap(err, fmt.Sprintf("failed to render kit"))
	// }
	// defer s.Kit.Close()

	go func() {
		for {
			key, err := tty.ReadRune()
			if err != nil {
				//TODO do something better than this
				fmt.Println("panic")
				panic(err)
			}

			note := s.switchKey(ctx, key, endCh)
			if note != nil {
				s.synth.NoteOn(note.Channel, note.Note, note.Velocity)
			}

			// e := <-uiEvents
			// switch e.ID {
			// case "q", "<C-c>":
			// 	return nil
			// }
		}
	}()

	switch <-endCh {
	case nil:
		close(endCh)
		break
	default:
		return <-endCh
	}

	return nil
}

func (s *Synth) switchKey(ctx context.Context, key rune, ch chan error) *midi.NoteMessage {
	switch key {
	case 98:
		return s.Kick()
	case 104:
		return s.HighHatClosed()
	case 106:
		return s.HighHatOpen()
	case 115:
		return s.Snare()
	case 99:
		return s.Clap()
	case 116:
		return s.TomHigh()
	case 121:
		return s.TomLow()
	case 3:
		//TODO cancel context here
		ch <- nil
	default:
		fmt.Println(key)
		// if keybind.IsPrintable(key) {
		// 	fmt.Printf("%c\n", key)
		// } else {
		// 	fmt.Printf("Ctrl+%c\n", '@'+key)
		// }
	}
	return nil
}
