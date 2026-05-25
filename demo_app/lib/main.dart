import 'package:flutter/widgets.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';

import 'src/demo_bootstrap_config.dart';
import 'src/demo_showcase_app.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await dotenv.load(isOptional: true);

  runApp(
    CortadoDemoShowcaseApp(
      initialConfig: DemoBootstrapConfig.fromSources(
        uri: Uri.base,
        env: dotenv.isInitialized ? dotenv.env : const <String, String>{},
      ),
    ),
  );
}
