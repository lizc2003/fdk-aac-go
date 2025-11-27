package fdkaac

/*
#include "deps/include/aacenc_lib.h"

AACENC_ERROR aacEncEncodeWrapped(const HANDLE_AACENCODER hAacEncoder,
		void* in, int inLen, int sampleBitDepth,
		void* out, int outLen, int* numOutBytes) {
	AACENC_ERROR err;
	AACENC_BufDesc inBuf = { 0 }, outBuf = { 0 };
	AACENC_InArgs inArgs = { 0 };
	AACENC_OutArgs outArgs = { 0 };
	int inIdentifier = IN_AUDIO_DATA;
	int inElemSize = sampleBitDepth / 8;
	int outIdentifier = OUT_BITSTREAM_DATA;
	int outElemSize = 1;

	inArgs.numInSamples = in ? inLen / inElemSize : -1;
	inBuf.numBufs = 1;
	inBuf.bufs = &in;
	inBuf.bufferIdentifiers = &inIdentifier;
	inBuf.bufSizes = &inLen;
	inBuf.bufElSizes = &inElemSize;
	outBuf.numBufs = 1;
	outBuf.bufs = &out;
	outBuf.bufferIdentifiers = &outIdentifier;
	outBuf.bufSizes = &outLen;
	outBuf.bufElSizes = &outElemSize;

	err = aacEncEncode(hAacEncoder, &inBuf, &outBuf, &inArgs, &outArgs);
	*numOutBytes = outArgs.numOutBytes;
	return err;
}
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

var encErrors = [...]error{
	C.AACENC_OK:                    nil,
	C.AACENC_INVALID_HANDLE:        errors.New("handle passed to function call was invalid"),
	C.AACENC_MEMORY_ERROR:          errors.New("memory allocation failed"),
	C.AACENC_UNSUPPORTED_PARAMETER: errors.New("parameter not available"),
	C.AACENC_INVALID_CONFIG:        errors.New("configuration not provided"),
	C.AACENC_INIT_ERROR:            errors.New("general initialization error"),
	C.AACENC_INIT_AAC_ERROR:        errors.New("AAC library initialization error"),
	C.AACENC_INIT_SBR_ERROR:        errors.New("SBR library initialization error"),
	C.AACENC_INIT_TP_ERROR:         errors.New("transport library initialization error"),
	C.AACENC_INIT_META_ERROR:       errors.New("meta data library initialization error"),
	C.AACENC_INIT_MPS_ERROR:        errors.New("MPS library initialization error"),
	C.AACENC_ENCODE_ERROR:          errors.New("the encoding process was interrupted by an unexpected error"),
	C.AACENC_ENCODE_EOF:            errors.New("end of file reached"),
}

// Encoder End Of File.
var EncEOF = encErrors[C.AACENC_ENCODE_EOF]

// getEncError safely converts C error code to Go error
func getEncError(errNo C.AACENC_ERROR) error {
	if int(errNo) >= 0 && int(errNo) < len(encErrors) {
		return encErrors[errNo]
	}
	return fmt.Errorf("unknown encoder error: %d", errNo)
}

// Bitrate Mode
type BitrateMode int

const (
	BitrateModeConstant BitrateMode = iota
	BitrateModeVeryLow
	BitrateModeLow
	BitrateModeMedium
	BitrateModeHigh
	BitrateModeVeryHigh
)

// Signaling Mode
type SignalingMode int

const (
	SignalingModeImplicitCompatible SignalingMode = iota
	SignalingModeExplicitCompatible
	SignalingModeExplicitHierarchical
)

// Meta Data Mode
type MetaDataMode int

const (
	MetaDataModeNone MetaDataMode = iota
	MetaDataModeDynamicRangeInfoOnly
	MetaDataModeDynamicRangeInfoAndAncillaryData
	MetaDataModeNoneAncillaryDataOnly
)

// AAC Encoder Config
type AacEncoderConfig struct {
	// Number of channels to be allocated.
	MaxChannels int
	// Audio object type.
	AOT AudioObjectType
	// Total encoder bitrate.
	Bitrate int
	// Bitrate mode.
	BitrateMode BitrateMode
	// Audio input data sampling rate.
	SampleRate int
	// Configure SBR independently of the chosen Audio Object Type.
	SbrMode SbrMode
	// Core encoder (AAC) audio frame length in samples.
	GranuleLength int
	// Set explicit channel mode. Channel mode must match with number of input channels.
	ChannelMode ChannelMode
	// Input audio data channel ordering scheme.
	ChannelOrder ChannelOrder
	// Controls activation of downsampled SBR.
	SbrRatio int
	// Controls the use of the afterburner feature.
	IsAfterBurner bool
	// Core encoder audio bandwidth.
	Bandwith int
	// Peak bitrate configuration parameter to adjust maximum bits per audio frame.
	PeakBitrate int
	// Transport type to be used.
	TransMux TransportType
	// Frame count period for sending in-band configuration buffers within LATM/LOAS transport layer.
	HeaderPeriod int
	// Signaling mode of the extension AOT.
	SignalingMode SignalingMode
	// Number of sub frames in a transport frame for LOAS/LATM or ADTS (default 1).
	TransportSubFrames int
	// AudioMuxVersion to be used for LATM.
	AudioMuxVersion int
	// Configure protection in transport layer.
	IsProtection bool
	// Constant ancillary data bitrate in bits/second.
	AncillaryBitrate int
	// Configure Meta Data.
	MetaDataMode MetaDataMode
}

// EncInfo provides some info about the encoder configuration.
type EncInfo struct {
	// Maximum number of encoder bitstream bytes within one frame.
	// Size depends on maximum number of supported channels in encoder instance.
	MaxOutBufBytes int
	// Maximum number of ancillary data bytes which can be
	// inserted into bitstream within one frame.
	MaxAncBytes int
	// Internal input buffer fill level in samples per channel.
	InBufFillLevel int
	// Number of input channels expected in encoding process.
	InputChannels int
	// Amount of input audio samples consumed each frame per channel,
	// depending on audio object type configuration.
	FrameLength int
	// Bytes per frame, including all channels
	FrameSize int
	// Codec delay in PCM samples/channel.
	NDelay int
	// Codec delay in PCM samples/channel.
	NDelayCore int
	// Configuration buffer in binary format as an AudioSpecificConfig or
	// StreamMuxConfig according to the selected transport type.
	ConfBuf []byte
}

// AAC Encoder
type AacEncoder struct {
	// private handler
	ph C.HANDLE_AACENCODER
	// info
	EncInfo
	prevDataBuffer []byte
}

func (enc *AacEncoder) Encode(in, out []byte) (n int, nFrames int, err error) {
	if enc == nil || enc.ph == nil {
		return 0, 0, errors.New("encoder not initialized")
	}

	szIn := len(in)
	szOut := len(out)

	if szIn == 0 {
		return 0, 0, errors.New("input buffer is empty")
	}
	if szOut < enc.EstimateOutBufBytes(szIn) {
		return 0, 0, errors.New("output buffer is too small")
	}

	frameSize := enc.FrameSize
	var nWrite C.int
	var inPtr, outPtr unsafe.Pointer

	prevLen := len(enc.prevDataBuffer)
	if prevLen > 0 {
		if (prevLen + szIn) >= frameSize {
			nFill := frameSize - prevLen
			enc.prevDataBuffer = append(enc.prevDataBuffer, in[:nFill]...)

			inPtr = unsafe.Pointer(&enc.prevDataBuffer[0])
			outPtr = unsafe.Pointer(&out[0])
			errNo := C.aacEncEncodeWrapped(enc.ph,
				inPtr, C.int(frameSize), C.int(SampleBitDepth),
				outPtr, C.int(szOut), &nWrite)
			if errNo != 0 {
				enc.prevDataBuffer = enc.prevDataBuffer[:prevLen]
				return 0, 0, getEncError(errNo)
			}
			enc.prevDataBuffer = enc.prevDataBuffer[:0]

			n = int(nWrite)
			nFrames = 1
			out = out[n:]
			in = in[nFill:]
			szIn = len(in)
			szOut = len(out)
		} else {
			enc.prevDataBuffer = append(enc.prevDataBuffer, in...)
			return 0, 0, nil
		}
	}

	if szIn < frameSize {
		enc.prevDataBuffer = append(enc.prevDataBuffer, in...)
		return n, nFrames, nil
	}

	for szIn >= frameSize {
		inPtr = unsafe.Pointer(&in[0])
		outPtr = unsafe.Pointer(&out[0])
		errNo := C.aacEncEncodeWrapped(enc.ph,
			inPtr, C.int(frameSize), C.int(SampleBitDepth),
			outPtr, C.int(szOut), &nWrite)
		if errNo != 0 {
			return 0, 0, getEncError(errNo)
		}
		nWr := int(nWrite)
		n += nWr
		nFrames++
		out = out[nWr:]
		in = in[frameSize:]
		szIn -= frameSize
		szOut -= nWr
	}

	if szIn > 0 {
		enc.prevDataBuffer = append(enc.prevDataBuffer, in...)
	}
	return n, nFrames, nil
}

// Flush
func (enc *AacEncoder) Flush(out []byte) (n int, nFrames int, err error) {
	szOut := len(out)
	if szOut < enc.EstimateOutBufBytes(0) {
		return 0, 0, errors.New("output buffer is too small")
	}

	var nWrite C.int
	var outPtr unsafe.Pointer

	if len(enc.prevDataBuffer) > 0 {
		var inPtr unsafe.Pointer
		inPtr = unsafe.Pointer(&enc.prevDataBuffer[0])
		outPtr = unsafe.Pointer(&out[0])
		errNo := C.aacEncEncodeWrapped(enc.ph,
			inPtr, C.int(len(enc.prevDataBuffer)), C.int(SampleBitDepth),
			outPtr, C.int(szOut), &nWrite)
		if errNo != 0 {
			return 0, 0, getEncError(errNo)
		}
		enc.prevDataBuffer = enc.prevDataBuffer[:0]

		n = int(nWrite)
		if n > 0 {
			nFrames = 1
			out = out[n:]
			szOut = len(out)
		}
	}

	for {
		outPtr = unsafe.Pointer(&out[0])
		errNo := C.aacEncEncodeWrapped(enc.ph,
			nil, 0, C.int(SampleBitDepth),
			outPtr, C.int(szOut), &nWrite)
		if errNo != 0 {
			if errNo == C.AACENC_ENCODE_EOF {
				return n, nFrames, nil
			}
			return 0, 0, getEncError(errNo)
		}

		nWr := int(nWrite)
		if nWr == 0 {
			return n, nFrames, nil
		}
		n += nWr
		nFrames++
		out = out[nWr:]
		szOut -= nWr
	}
}

// Close
func (enc *AacEncoder) Close() error {
	if enc == nil || enc.ph == nil {
		return nil
	}
	err := getEncError(C.aacEncClose(&enc.ph))
	enc.ph = nil
	return err
}

func (enc *AacEncoder) EstimateOutBufBytes(inBytes int) int {
	// The maximum packet size is 768 bytes per channel.
	nFrames := inBytes/enc.FrameSize + 1 + 3
	return nFrames * enc.MaxOutBufBytes
}

// Create AAC Encoder
func CreateAacEncoder(config *AacEncoderConfig) (enc *AacEncoder, err error) {
	config = populateEncConfig(config)

	// Validate configuration
	if config.MaxChannels <= 0 || config.MaxChannels > 8 {
		return nil, fmt.Errorf("invalid MaxChannels: %d (must be 1-8)", config.MaxChannels)
	}
	if config.SampleRate <= 0 {
		return nil, fmt.Errorf("invalid SampleRate: %d", config.SampleRate)
	}
	if config.MetaDataMode != MetaDataModeNone {
		return nil, errors.New("metadata mode is not supported yet")
	}

	enc = &AacEncoder{}

	var errNo C.AACENC_ERROR
	if errNo = C.aacEncOpen(&enc.ph, 0, C.uint(config.MaxChannels)); errNo != C.AACENC_OK {
		return nil, getEncError(errNo)
	}

	defer func() {
		if errNo != C.AACENC_OK {
			C.aacEncClose(&enc.ph)
		}
	}()

	if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_AOT,
		C.uint(config.AOT)); errNo != C.AACENC_OK {
		return nil, getEncError(errNo)
	}
	if config.BitrateMode == BitrateModeConstant {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_BITRATE,
			C.uint(config.Bitrate)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	} else {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_BITRATEMODE,
			C.uint(config.BitrateMode)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}
	if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_SAMPLERATE,
		C.uint(config.SampleRate)); errNo != C.AACENC_OK {
		return nil, getEncError(errNo)
	}
	if config.SbrMode != SbrModeDefault {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_SBR_MODE,
			C.uint(config.SbrMode-1)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}
	if config.GranuleLength != 0 {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_GRANULE_LENGTH,
			C.uint(config.GranuleLength)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}
	if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_CHANNELMODE,
		C.uint(config.ChannelMode)); errNo != C.AACENC_OK {
		return nil, getEncError(errNo)
	}
	if config.ChannelOrder != ChannelOrderMpeg {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_CHANNELORDER,
			C.uint(config.ChannelOrder)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}
	if config.SbrRatio != 0 {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_SBR_RATIO,
			C.uint(config.SbrRatio)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}
	if config.IsAfterBurner {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_AFTERBURNER,
			C.uint(1)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}
	if config.Bandwith > 0 {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_BANDWIDTH,
			C.uint(config.Bandwith)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}
	if config.PeakBitrate > 0 {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_PEAK_BITRATE,
			C.uint(config.PeakBitrate)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}
	if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_TRANSMUX,
		C.uint(config.TransMux)); errNo != C.AACENC_OK {
		return nil, getEncError(errNo)
	}
	if config.HeaderPeriod > 0 {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_HEADER_PERIOD,
			C.uint(config.HeaderPeriod)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}
	if config.SignalingMode != SignalingModeImplicitCompatible {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_SIGNALING_MODE,
			C.uint(config.SignalingMode)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}
	if config.TransportSubFrames > 0 {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_TPSUBFRAMES,
			C.uint(config.TransportSubFrames)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}
	if config.AudioMuxVersion > 0 {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_AUDIOMUXVER,
			C.uint(config.AudioMuxVersion)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}
	if config.IsProtection {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_PROTECTION,
			C.uint(1)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}
	if config.AncillaryBitrate > 0 {
		if errNo = C.aacEncoder_SetParam(enc.ph, C.AACENC_ANCILLARY_BITRATE,
			C.uint(config.AncillaryBitrate)); errNo != C.AACENC_OK {
			return nil, getEncError(errNo)
		}
	}

	if errNo = C.aacEncEncode(enc.ph, nil, nil, nil, nil); errNo != C.AACENC_OK {
		return nil, getEncError(errNo)
	}

	if errNo = enc.getInfo(); errNo != C.AACENC_OK {
		return nil, getEncError(errNo)
	}

	enc.prevDataBuffer = make([]byte, 0, enc.FrameSize)
	return enc, nil
}

func (enc *AacEncoder) getInfo() C.AACENC_ERROR {
	info := C.AACENC_InfoStruct{}
	if errNo := C.aacEncInfo(enc.ph, &info); errNo != C.AACENC_OK {
		return errNo
	}

	enc.MaxOutBufBytes = int(info.maxOutBufBytes)
	enc.MaxAncBytes = int(info.maxAncBytes)
	enc.InBufFillLevel = int(info.inBufFillLevel)
	enc.InputChannels = int(info.inputChannels)
	enc.FrameLength = int(info.frameLength)
	enc.NDelay = int(info.nDelay)
	enc.NDelayCore = int(info.nDelayCore)
	enc.ConfBuf = C.GoBytes(unsafe.Pointer(&info.confBuf[0]), C.int(info.confSize))
	enc.FrameSize = enc.FrameLength * enc.InputChannels * SampleBitDepth / 8
	return C.AACENC_OK
}

func populateEncConfig(c *AacEncoderConfig) *AacEncoderConfig {
	if c == nil {
		c = &AacEncoderConfig{}
	}
	if c.MaxChannels == 0 {
		c.MaxChannels = defaultMaxChannels
	}
	if c.AOT == 0 {
		c.AOT = defaultAOT
	}
	if c.ChannelMode == 0 {
		if c.MaxChannels <= 7 {
			c.ChannelMode = ChannelMode(c.MaxChannels)
		}
	}
	if c.SampleRate == 0 {
		c.SampleRate = defaultSamplerate
	}
	if c.Bitrate == 0 {
		c.Bitrate = defaultBitrate
	}

	return c
}
