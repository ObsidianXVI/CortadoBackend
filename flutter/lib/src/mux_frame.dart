import 'dart:typed_data';

const int muxFrameHeaderLength = 7;
const int muxTerminalChannelId = 0x0001;
const int muxMessageTypeData = 0x01;
const int muxMessageTypeOpen = 0x02;
const int muxMessageTypeClose = 0x03;
const int muxMessageTypeError = 0x04;
const int muxMessageTypePing = 0xFF;

class MuxFrame {
  const MuxFrame(this.channelId, this.messageType, this.payload);

  final int channelId;
  final int messageType;
  final Uint8List payload;

  Uint8List encode() {
    final byteData = ByteData(muxFrameHeaderLength + payload.length);
    byteData.setUint16(0, channelId, Endian.big);
    byteData.setUint8(2, messageType);
    byteData.setUint32(3, payload.length, Endian.big);

    final bytes = byteData.buffer.asUint8List();
    bytes.setRange(
        muxFrameHeaderLength, muxFrameHeaderLength + payload.length, payload);
    return bytes;
  }

  static MuxFrame decode(Uint8List bytes) {
    if (bytes.length < muxFrameHeaderLength) {
      throw const FormatException('Frame too short.');
    }

    final byteData = ByteData.sublistView(bytes);
    final payloadLength = byteData.getUint32(3, Endian.big);
    if (bytes.length != muxFrameHeaderLength + payloadLength) {
      throw const FormatException('Frame length mismatch.');
    }

    return MuxFrame(
      byteData.getUint16(0, Endian.big),
      byteData.getUint8(2),
      // View, not copy — do not mutate source bytes after decode.
      Uint8List.sublistView(
          bytes, muxFrameHeaderLength, muxFrameHeaderLength + payloadLength),
    );
  }
}
