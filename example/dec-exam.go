package main

import (
	"fmt"
	"github.com/lizc2003/audio-fdkaac"
	"io"
	"os"
)

func main() {
	decoder, err := fdkaac.CreateAacDecoder(&fdkaac.AacDecoderConfig{
		TransportFmt: fdkaac.TtMp4Adts,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer decoder.Close()

	aacFile, err := os.Open("samples/sample.aac")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer aacFile.Close()

	pcmBuf := make([]byte, decoder.EstimateOutBufBytes())
	totalFrames := 0
	totalBytes := 0
	chunk := make([]byte, 2048)
	var residue []byte

	for {
		n, readErr := aacFile.Read(chunk)
		if n > 0 {
			residue = append(residue, chunk[:n]...)

			inBuf := residue
			for {
				decodedN, nFrames, rest, decErr := decoder.Decode(inBuf, pcmBuf)
				if decErr != nil {
					fmt.Println(decErr)
					return
				}

				if decodedN == 0 {
					// Not enough data to decode a frame, need more input
					residue = residue[:copy(residue, rest)]
					break
				}

				totalBytes += decodedN
				totalFrames += nFrames
				inBuf = rest

				if len(inBuf) == 0 {
					residue = residue[:0]
					break
				}
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			fmt.Println(readErr)
			return
		}
	}

	fmt.Printf("Decoded %d bytes of PCM data, totalFrames: %d\n", totalBytes, totalFrames)
}
