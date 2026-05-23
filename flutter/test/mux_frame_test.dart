import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('MuxFrame', () {
    test('encodes and decodes a frame round trip', () {
      final frame = MuxFrame(
        0x0001,
        0x01,
        Uint8List.fromList(<int>[0x41, 0x42, 0x43]),
      );

      final encoded = frame.encode();
      final decoded = MuxFrame.decode(encoded);

      expect(decoded.channelId, 0x0001);
      expect(decoded.messageType, 0x01);
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

      expect(frame.channelId, 0x0001);
      expect(frame.messageType, 0x01);
      expect(frame.payload, orderedEquals(<int>[0x41, 0x42, 0x43]));
    });

    test('rejects a truncated frame', () {
      expect(
        () => MuxFrame.decode(Uint8List.fromList(<int>[0x00, 0x01, 0x01])),
        throwsFormatException,
      );
    });
  });
}
