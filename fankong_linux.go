package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

const (
	ExitOK    = 0
	ExitError = 1
)

const (
	NVIDIA_SETTINGS_EXECUTABLE_NAME = "nvidia-settings"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(ExitError)
	}
	os.Exit(ExitOK)
}

func run(args []string) error {
	config := NewConfig()

	flags := flag.NewFlagSet("fankong", flag.ContinueOnError)
	flags.StringVar(&config.Display, "display", "", "")
	flags.StringVar(&config.Xauthority, "xauthority", "", "")
	flags.DurationVar(&config.Interval, "interval", 30*time.Second, "")
	flags.IntVar(&config.TargetTemp, "target-temp", 60, "")
	flags.UintVar(&config.MinFanSpeed, "min-fan-speed", 30, "")
	flags.UintVar(&config.MaxFanSpeed, "max-fan-speed", 100, "")

	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	argv := flags.Args()
	if len(argv) > 0 {
		return fmt.Errorf("cannot pass argument")
	}

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer stop()

	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	app, err := NewApp(config)
	if err != nil {
		return err
	}

	// first time
	if err = app.Run(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err = app.Run(ctx); err != nil {
				return err
			}
		}
	}
}

type App struct {
	config *Config
}

func NewApp(config *Config) (*App, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &App{config}, nil
}

func (app *App) Run(ctx context.Context) error {
	temp, err := app.getGPUCoreTemp(ctx)
	if err != nil {
		return err
	}
	fan, err := app.getGPUTargetFanSpeed(ctx)
	if err != nil {
		return err
	}
	log.Printf("[info] %dC / %d%%\n", temp, fan)
	if temp > app.config.TargetTemp && fan < app.config.MaxFanSpeed {
		app.setGPUTargetFanSpeed(ctx, fan+1)
	}
	if temp < app.config.TargetTemp && fan > app.config.MinFanSpeed {
		app.setGPUTargetFanSpeed(ctx, fan-1)
	}
	return nil
}

func (app *App) getGPUCoreTemp(ctx context.Context) (int, error) {
	res, err := app.execNvidiaSettings(ctx, "-q", "[gpu:0]/GPUCoreTemp", "-t")
	if err != nil {
		return 0, err
	}

	out := strings.TrimSuffix(string(res), "\n")
	value, err := strconv.ParseInt(out, 10, 64)
	if err != nil {
		return 0, err
	}

	return int(value), nil
}

func (app *App) getGPUTargetFanSpeed(ctx context.Context) (uint, error) {
	res, err := app.execNvidiaSettings(ctx, "-q", "[fan:0]/GPUTargetFanSpeed", "-t")
	if err != nil {
		return 0, err
	}

	out := strings.TrimSuffix(string(res), "\n")
	value, err := strconv.ParseInt(out, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(value), nil
}

func (app *App) setGPUTargetFanSpeed(ctx context.Context, speed uint) error {
	_, err := app.execNvidiaSettings(ctx,
		"-a",
		"[fan:0]/GPUFanControlState=1",
		"-a",
		fmt.Sprintf("[fan:0]/GPUTargetFanSpeed=%d", speed),
	)
	return err
}

func (app *App) queryIntValueFromNvidiaSettings(ctx context.Context, query string) (int, error) {
	res, err := app.execNvidiaSettings(ctx, "-q", query, "-t")
	if err != nil {
		return 0, err
	}

	out := strings.TrimSuffix(string(res), "\n")
	value, err := strconv.ParseInt(out, 10, 64)
	if err != nil {
		return 0, err
	}

	return int(value), nil
}

func (app *App) execNvidiaSettings(ctx context.Context, arg ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, NVIDIA_SETTINGS_EXECUTABLE_NAME, arg...)

	// set env
	cmd.Env = os.Environ()
	cmd.Env = append(
		cmd.Env,
		fmt.Sprintf("DISPLAY=%s", app.config.Display),
		fmt.Sprintf("XAUTHORITY=%s", app.config.Xauthority),
	)

	// set stdout and stderr handler
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %w", stderr.String(), err)
	}

	return stdout.Bytes(), nil
}
