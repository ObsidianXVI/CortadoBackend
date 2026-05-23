import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('MuxFrame', () {
    test('encodes and decodes a frame round trip', () {
      final frame = MuxFrame(
        muxTerminalChannelId,
        muxMessageTypeData,
        Uint8List.fromList(<int>[0x41, 0x42, 0x43]),
      );

      final encoded = frame.encode();
      final decoded = MuxFrame.decode(encoded);

      expect(decoded.channelId, muxTerminalChannelId);
      expect(decoded.messageType, muxMessageTypeData);
      expect(decoded.payload, orderedEquals(<int>[0x41, 0x42, 0x43]));
    });

    test('decodes the hard-coded Go frame bytes', () {
      final frame = MuxFrame.decode(
        Uint8List.fromList(<int>[
          0x00,
          0x01,
          0x01,
          0x00,
          0x00,
          0x00,
          0x03,
          0x41,
          0x42,
          0x43,
        ]),
      );

      expect(frame.channelId, muxTerminalChannelId);
      expect(frame.messageType, muxMessageTypeData);
      expect(frame.payload, orderedEquals(<int>[0x41, 0x42, 0x43]));
    });

    test('rejects a truncated frame', () {
      expect(
        () => MuxFrame.decode(Uint8List.fromList(<int>[0x00, 0x01, 0x01])),
        throwsFormatException,
      );
    });

    test('keeps payload as a view over the source bytes', () {
      final encoded = Uint8List.fromList(<int>[
        0x00,
        0x01,
        0x01,
        0x00,
        0x00,
        0x00,
        0x03,
        0x41,
        0x42,
        0x43,
      ]);

      final frame = MuxFrame.decode(encoded);
      encoded[7] = 0x5A;

      expect(frame.payload, orderedEquals(<int>[0x5A, 0x42, 0x43]));
    });
  });
}
