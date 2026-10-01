/// Base URL of the Go API.
///
/// Override at build time:
///   flutter run --dart-define=API_URL=http://192.168.1.10:8080
/// The default targets the Android emulator's alias for the host machine.
const String apiUrl = String.fromEnvironment('API_URL', defaultValue: 'http://10.0.2.2:8080');
