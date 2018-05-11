package logging

import (
    "io"

    "go.uber.org/zap/zapcore"
)

type WriteSyncer = zapcore.WriteSyncer

// Lock is an convenient function to convert from generic golang io.writer.
func Lock(w io.Writer) WriteSyncer {
    // Use NoOp Sync for protection.
    writer := zapcore.AddSync(w)
    return zapcore.Lock(writer)
}