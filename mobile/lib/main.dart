import 'package:flutter/material.dart';
import 'package:intl/date_symbol_data_local.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';

import 'api/api_client.dart';
import 'screens/auth_screen.dart';
import 'screens/home_shell.dart';
import 'state/session.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  Intl.defaultLocale = 'fr_FR';
  await initializeDateFormatting('fr_FR');
  final session = Session(ApiClient());
  session.restore();
  runApp(ChangeNotifierProvider.value(value: session, child: const MicroJobApp()));
}

class MicroJobApp extends StatelessWidget {
  const MicroJobApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'MicroJob',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF4F46E5)),
        useMaterial3: true,
        cardTheme: const CardThemeData(elevation: 0.5),
        inputDecorationTheme: const InputDecorationTheme(border: OutlineInputBorder()),
      ),
      home: Consumer<Session>(
        builder: (context, s, _) {
          if (s.loading) return const Scaffold(body: Center(child: CircularProgressIndicator()));
          return s.user == null ? const AuthScreen() : const HomeShell();
        },
      ),
    );
  }
}
