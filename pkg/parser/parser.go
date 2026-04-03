package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/yourusername/qrgen/pkg/types"
)

// Flag name constants used by ParseArgs.
const (
	flagQuantity  = "--quantity"
	flagSize      = "--size"
	flagInfo      = "--info"
	flagURL       = "--url"
	flagFormat    = "--format"
	flagStorage   = "--storage"
	flagChunkSize = "--chunk-size"
	flagSendEmail = "--send-email"
	flagEmailTo   = "--email-to"
	flagProjectID = "--project-id"
	flagBucket    = "--bucket"
)

// ParseTime formats a duration given in milliseconds into a human friendly string.
// The function mirrors the behavior of the original implementation: microseconds,
// milliseconds, seconds, minutes, hours, days.
func ParseTime(durationMs float64) string {
	// For microseconds (< 1 ms)
	if durationMs < 1 {
		return fmt.Sprintf("%.2f μs", durationMs*1000)
	}

	// For milliseconds (< 1000 ms)
	if durationMs < 1000 {
		return fmt.Sprintf("%.2f ms", durationMs)
	}

	// For seconds (< 60 s)
	seconds := durationMs / 1000.0
	if seconds < 60 {
		return fmt.Sprintf("%.2f s", seconds)
	}

	// For minutes (< 60 min)
	minutes := seconds / 60.0
	if minutes < 60 {
		minutesInt := int(minutes)
		secondsInt := int((minutes - float64(minutesInt)) * 60)
		return fmt.Sprintf("%d min %d s", minutesInt, secondsInt)
	}

	// For hours (< 24 h)
	hours := minutes / 60.0
	if hours < 24 {
		hoursInt := int(hours)
		minutesInt := int((hours - float64(hoursInt)) * 60)
		return fmt.Sprintf("%d h %d min", hoursInt, minutesInt)
	}

	// Days
	return fmt.Sprintf("%.2f days", hours/24.0)
}

// ParseArgs parses CLI-style arguments into an Argument object.
// Supported flags:
//
//	--quantity=N        (required, integer > 0)
//	--url=URL           (required)
//	--size=N            (optional, integer, default 50)
//	--info=TEXT         (optional)
//	--format=pdf        (optional, only 'pdf' is accepted as valid output format)
//	--storage=local     (optional, default 'local')
//	--version or -v     (causes ParseArgs to return error("version"))
//	--help or -h        (causes ParseArgs to return error("help"))
//
// Returns (*types.Argument, nil) on success.
// Returns (nil, error) on failure. The special errors "help" and "version" are
// returned when the corresponding flags are present so callers can show usage/version.
func ParseArgs(args []string) (*types.Argument, error) {
	// Defaults to mirror original behavior
	arg := &types.Argument{
		Quantity: 0,
		Info:     "",
		Size:     50,
		URL:      "",
		Format:   types.PDF,
		Storage:  types.LOCAL,
	}

	for _, a := range args {
		// allow flags either as "--flag=value" or "--flag" (for help/version)
		parts := strings.SplitN(a, "=", 2)
		key := parts[0]
		val := ""
		if len(parts) == 2 {
			val = parts[1]
		}

		switch key {
		case "--version", "-v":
			return nil, errors.New("version")
		case "--help", "-h":
			return nil, errors.New("help")
		case flagQuantity:
			if val == "" {
				return nil, fmt.Errorf("%s requires a value", flagQuantity)
			}
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid %s value: %w", flagQuantity, err)
			}
			arg.Quantity = n
		case flagSize:
			if val == "" {
				return nil, fmt.Errorf("%s requires a value", flagSize)
			}
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid %s value: %w", flagSize, err)
			}
			arg.Size = n
		case flagInfo:
			arg.Info = val
		case flagURL:
			arg.URL = val
		case flagFormat:
			if val == "" {
				return nil, fmt.Errorf("%s requires a value", flagFormat)
			}
			f, err := types.FormatFromString(val)
			if err != nil {
				return nil, fmt.Errorf("invalid format: %w", err)
			}
			arg.Format = f
		case flagStorage:
			if val == "" {
				return nil, fmt.Errorf("%s requires a value", flagStorage)
			}
			s, err := types.StorageFromString(val)
			if err != nil {
				return nil, fmt.Errorf("invalid storage: %w", err)
			}
			arg.Storage = s
		case flagChunkSize:
			if val == "" {
				return nil, fmt.Errorf("%s requires a value", flagChunkSize)
			}
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid %s value: %w", flagChunkSize, err)
			}
			if n < 1 {
				return nil, fmt.Errorf("%s must be greater than 0", flagChunkSize)
			}
			arg.ChunkSize = n
		case flagSendEmail:
			// --send-email is a boolean flag; presence means true, optional =true/=false value
			if val == "" || strings.EqualFold(val, "true") || val == "1" {
				arg.SendEmail = true
			} else if strings.EqualFold(val, "false") || val == "0" {
				arg.SendEmail = false
			} else {
				return nil, fmt.Errorf("invalid %s value %q: expected true/false", flagSendEmail, val)
			}
		case flagEmailTo:
			if val == "" {
				return nil, fmt.Errorf("%s requires a value", flagEmailTo)
			}
			arg.EmailTo = val
		case flagProjectID:
			if val == "" {
				return nil, fmt.Errorf("%s requires a value", flagProjectID)
			}
			arg.ProjectID = val
		case flagBucket:
			if val == "" {
				return nil, fmt.Errorf("%s requires a value", flagBucket)
			}
			arg.Bucket = val
		default:
			// ignore unknown args for forward compatibility
		}
	}

	// validate required arguments
	if arg.Quantity < 1 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}
	if strings.TrimSpace(arg.URL) == "" {
		return nil, fmt.Errorf("url is required")
	}
	// For parity with original CLI, only PDF is accepted as output format
	if arg.Format != types.PDF {
		return nil, fmt.Errorf("unsupported format %s", arg.Format.String())
	}
	// email-to is required when --send-email is set
	if arg.SendEmail && strings.TrimSpace(arg.EmailTo) == "" {
		return nil, fmt.Errorf("%s is required when %s is set", flagEmailTo, flagSendEmail)
	}

	return arg, nil
}
