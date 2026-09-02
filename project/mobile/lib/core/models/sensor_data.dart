import 'package:equatable/equatable.dart';

String _stringValue(dynamic value) => value?.toString() ?? '';

double _doubleValue(dynamic value, [double fallback = 0]) {
  if (value == null) {
    return fallback;
  }
  if (value is num) {
    return value.toDouble();
  }
  return double.tryParse(value.toString()) ?? fallback;
}

bool _boolValue(dynamic value, {bool fallback = false}) {
  if (value == null) {
    return fallback;
  }
  if (value is bool) {
    return value;
  }
  final text = value.toString().trim().toLowerCase();
  return text == 'true' || text == '1' || text == 'yes';
}

DateTime _parseTimestamp(dynamic raw) {
  if (raw == null) {
    return DateTime.now();
  }

  if (raw is DateTime) {
    return raw;
  }

  if (raw is num) {
    final value = raw.toInt();
    // Treat 10-digit values as seconds and 13-digit values as milliseconds.
    if (value < 100000000000) {
      return DateTime.fromMillisecondsSinceEpoch(value * 1000);
    }
    return DateTime.fromMillisecondsSinceEpoch(value);
  }

  final text = raw.toString().trim();
  if (text.isEmpty) {
    return DateTime.now();
  }

  final numeric = int.tryParse(text);
  if (numeric != null) {
    if (numeric < 100000000000) {
      return DateTime.fromMillisecondsSinceEpoch(numeric * 1000);
    }
    return DateTime.fromMillisecondsSinceEpoch(numeric);
  }

  return DateTime.tryParse(text) ?? DateTime.now();
}

class SensorData extends Equatable {
  final String deviceId;
  final double waterTemperature;
  final bool temperatureValid;
  final double feedLevel;
  final double batteryLevel;
  final double solarVoltage;
  final bool isSolarCharging;
  final int signalStrength;
  final String connectionType;
  final DateTime timestamp;

  const SensorData({
    required this.deviceId,
    required this.waterTemperature,
    required this.temperatureValid,
    required this.feedLevel,
    required this.batteryLevel,
    required this.solarVoltage,
    required this.isSolarCharging,
    required this.signalStrength,
    required this.connectionType,
    required this.timestamp,
  });

  factory SensorData.fromJson(Map<String, dynamic> json) {
    final timestampRaw = json['timestamp'] ?? json['created_at'];
    final waterTemperature = _doubleValue(
      json['water_temperature'] ?? json['temperature'],
    );
    return SensorData(
      deviceId: json['device_id'] ?? '',
      waterTemperature: waterTemperature,
      temperatureValid: _boolValue(
        json['temperature_valid'],
        fallback: waterTemperature > 0,
      ),
      feedLevel: _doubleValue(
        json['feed_level'] ?? json['weight_percentage'] ?? 0,
      ),
      batteryLevel: _doubleValue(json['battery_level']),
      solarVoltage: _doubleValue(json['solar_voltage']),
      isSolarCharging:
          json['is_solar_charging'] ??
          ((json['power_source'] ?? '').toString().toLowerCase() == 'solar'),
      signalStrength: json['signal_strength'] ?? json['cellular_signal'] ?? 0,
      connectionType:
          json['connection_type'] ?? json['power_source'] ?? 'unknown',
      timestamp: _parseTimestamp(timestampRaw),
    );
  }

  Map<String, dynamic> toJson() => {
    'device_id': deviceId,
    'water_temperature': waterTemperature,
    'temperature_valid': temperatureValid,
    'feed_level': feedLevel,
    'battery_level': batteryLevel,
    'solar_voltage': solarVoltage,
    'is_solar_charging': isSolarCharging,
    'signal_strength': signalStrength,
    'connection_type': connectionType,
    'timestamp': timestamp.toIso8601String(),
  };

  @override
  List<Object?> get props => [
    deviceId,
    waterTemperature,
    temperatureValid,
    feedLevel,
    batteryLevel,
    timestamp,
  ];
}

class SensorHistory {
  final String sensorType;
  final List<SensorDataPoint> dataPoints;
  final double? min;
  final double? max;
  final double? average;

  SensorHistory({
    required this.sensorType,
    required this.dataPoints,
    this.min,
    this.max,
    this.average,
  });

  factory SensorHistory.fromJson(Map<String, dynamic> json) {
    return SensorHistory(
      sensorType: json['sensor_type'] ?? '',
      dataPoints:
          (json['data_points'] as List?)
              ?.map((e) => SensorDataPoint.fromJson(e))
              .toList() ??
          [],
      min: json['min']?.toDouble(),
      max: json['max']?.toDouble(),
      average: json['average']?.toDouble(),
    );
  }
}

class SensorDataPoint {
  final double value;
  final DateTime timestamp;

  SensorDataPoint({required this.value, required this.timestamp});

  factory SensorDataPoint.fromJson(Map<String, dynamic> json) {
    return SensorDataPoint(
      value: (json['value'] ?? 0).toDouble(),
      timestamp: DateTime.parse(
        json['timestamp'] ?? DateTime.now().toIso8601String(),
      ),
    );
  }
}

enum AlertSeverity { info, warning, critical }

class DeviceAlert extends Equatable {
  final String id;
  final String deviceId;
  final String title;
  final String message;
  final AlertSeverity severity;
  final String alertType;
  final bool isRead;
  final DateTime createdAt;

  const DeviceAlert({
    required this.id,
    required this.deviceId,
    required this.title,
    required this.message,
    required this.severity,
    required this.alertType,
    required this.isRead,
    required this.createdAt,
  });

  factory DeviceAlert.fromJson(Map<String, dynamic> json) {
    final alertType = json['alert_type'] ?? json['type'] ?? '';

    return DeviceAlert(
      id: _stringValue(json['id']),
      deviceId: json['device_id'] ?? '',
      title: json['title'] ?? _humanizeAlertType(alertType),
      message: json['message'] ?? '',
      severity: _parseSeverity(json['severity']),
      alertType: alertType,
      isRead: json['is_read'] ?? false,
      createdAt: DateTime.parse(
        json['created_at'] ??
            json['timestamp'] ??
            DateTime.now().toIso8601String(),
      ),
    );
  }

  static AlertSeverity _parseSeverity(String? severity) {
    switch (severity) {
      case 'critical':
      case 'high':
        return AlertSeverity.critical;
      case 'warning':
      case 'medium':
        return AlertSeverity.warning;
      default:
        return AlertSeverity.info;
    }
  }

  static String _humanizeAlertType(String alertType) {
    if (alertType.isEmpty) {
      return 'Alert';
    }

    return alertType
        .split('_')
        .map(
          (part) =>
              part.isEmpty
                  ? part
                  : '${part[0].toUpperCase()}${part.substring(1).toLowerCase()}',
        )
        .join(' ');
  }

  @override
  List<Object?> get props => [id, deviceId, title, severity, isRead, createdAt];
}
