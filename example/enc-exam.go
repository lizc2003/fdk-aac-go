package main

import (
	"fmt"
	"github.com/lizc2003/audio-fdkaac"
	"os"
)

func main() {
	encoder, err := fdkaac.CreateAacEncoder(&fdkaac.AacEncoderConfig{
		TransMux:    fdkaac.TtMp4Adts,
		SampleRate:  44100,
		MaxChannels: 2,
		Bitrate:     128000,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer encoder.Close()

	inBuf, err := os.ReadFile("samples/sample.pcm")
	if err != nil {
		fmt.Println(err)
		return
	}

	outBuf := make([]byte, encoder.EstimateOutBufBytes(len(inBuf)))
	n, nFrames, err := encoder.Encode(inBuf, outBuf)
	if err != nil {
		fmt.Println(err)
		return
	}

	n2, nFrames2, err := encoder.Flush(outBuf[n:])
	if err != nil {
		fmt.Println(err)
		return
	}

	outBuf = outBuf[:n+n2]
	fmt.Printf("total bytes:%d, frames:%d\n", len(outBuf), nFrames+nFrames2)
	os.WriteFile("output.aac", outBuf, 0644)
}
