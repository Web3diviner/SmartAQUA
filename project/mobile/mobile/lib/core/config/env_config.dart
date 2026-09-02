/// Environment configuration for the SmartAqua mobile app.
///
/// Values are loaded from compile-time environment variables using --dart-define.
///
/// Build commands:
/// ```bash
/// # Development
/// flutter build apk --dart-define-from-file=.env
///
/// # Or individual defines
/// flutter build apk \
///   --dart-define=API_BASE_URL=https://smartaqua.onrender.com/api/v1 \
///   --dart-define=MQTT_HOST=your-broker-host \
///   --dart-define=MQTT_PORT=8883 \
///   --dart-define=MQTT_USERNAME=your-broker-username \
///   --dart-define=MQTT_PASSWORD=your-broker-password
/// ```
class EnvConfig {
  // Private constructor
  EnvConfig._();

  /// API base URL
  static const String apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'https://smartaqua.onrender.com/api/v1',
  );

  /// MQTT broker host
  static const String mqttHost = String.fromEnvironment(
    'MQTT_HOST',
    defaultValue: '',
  );

  /// MQTT broker port
  static const int mqttPort = int.fromEnvironment(
    'MQTT_PORT',
    defaultValue: 8883,
  );

  /// Optional MQTT username for brokers that do not use JWT auth
  static const String mqttUsername = String.fromEnvironment(
    'MQTT_USERNAME',
    defaultValue: '',
  );

  /// Optional MQTT password for brokers that do not use JWT auth
  static const String mqttPassword = String.fromEnvironment(
    'MQTT_PASSWORD',
    defaultValue: '',
  );

  /// Primary certificate fingerprint for TLS pinning
  static const String certFingerprint1 = String.fromEnvironment(
    'CERT_FINGERPRINT_1',
    defaultValue: '',
  );

  /// Backup certificate fingerprint for TLS pinning
  static const String certFingerprint2 = String.fromEnvironment(
    'CERT_FINGERPRINT_2',
    defaultValue: '',
  );

  /// API domain for certificate validation
  static const String apiDomain = String.fromEnvironment(
    'API_DOMAIN',
    defaultValue: 'smartaqua.onrender.com',
  );

  /// Debug mode flag
  static const bool debugMode = bool.fromEnvironment(
    'DEBUG_MODE',
    defaultValue: false,
  );

  /// Check if running in development mode
  static bool get isDevelopment =>
      const bool.fromEnvironment('dart.vm.product') == false;

  /// Get list of pinned certificate fingerprints
  static List<String> get pinnedCertificates {
    final certs = <String>[];
    if (certFingerprint1.isNotEmpty) {
      certs.add('sha256/$certFingerprint1');
    }
    if (certFingerprint2.isNotEmpty) {
      certs.add('sha256/$certFingerprint2');
    }
    return certs;
  }
}
