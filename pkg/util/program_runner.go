package util

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/rs/zerolog"
)

type ProgramRunner struct {
	MaxRetries    int
	InterRunDelay time.Duration

	Program     string
	Args        []string
	SysProcAttr syscall.SysProcAttr

	l zerolog.Logger
}

func (p *ProgramRunner) launchProgram(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, p.Program, p.Args...)

	cmd.Stderr = NewProcessLogWrapper(p.l, zerolog.ErrorLevel)
	cmd.Stdout = NewProcessLogWrapper(p.l, zerolog.InfoLevel)
	cmd.SysProcAttr = &p.SysProcAttr
	cmd.WaitDelay = time.Second * 5

	return cmd.Run()
}

func (p *ProgramRunner) Run(component string, ctx context.Context) error {
	p.l = log.NewLogger(component, nil)
	tries := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			p.l.Info().Msgf("Launching %v", p.String())
			err := p.launchProgram(ctx)
			if err != nil {
				p.l.Error().Err(err).Msg("Program exited with error")
			}
			if tries >= p.MaxRetries {
				p.l.Error().Msg("Program failed too many times, exiting")
				return errors.Join(err, fmt.Errorf("max retries reached"))
			} else {
				tries++
			}
			if p.InterRunDelay > 0 {
				p.l.Info().Msgf("Waiting %v for next run", p.InterRunDelay.String())
				time.Sleep(p.InterRunDelay)
			} else {
				p.l.Info().Msg("No delay between runs, launching immediately")
			}
		}
	}
}

func (p *ProgramRunner) String() string {
	return fmt.Sprintf("%v %v", p.Program, strings.Join(p.Args, " "))
}
