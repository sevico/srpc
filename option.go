package srpc

import (
	"srpc/codec"
	"time"
)

const MagicNumber = 0x3beef
type Option struct{
	MagicNumber int
	CodecType codec.Type
	ConnectTimeout time.Duration
	HandleTimeout time.Duration
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType: codec.GobType,
	ConnectTimeout: time.Second * 10,
}