import 'package:equatable/equatable.dart';

String _stringValue(dynamic value) => value?.toString() ?? '';

DateTime? _dateTimeValue(dynamic value) {
  if (value is String && value.isNotEmpty) {
    return DateTime.tryParse(value);
  }
  return null;
}

class Device extends Equatable {
  final String id;
  final String name;
  final String serialNumber;
  final bool isOnline;
  final DateTime? lastSeen;
  final DeviceStatus status;
  final DeviceConfig config;

  const Device({
    required this.id,
    required this.name,
    required this.serialNumber,
    required this.isOnline,
    this.lastSeen,
    required this.status,
    required this.config,
  });

  factory Device.fromJson(Map<String, dynamic> json) {
    final lastSeen = _dateTimeValue(json['last_seen']);
    final statusJson =
        json['status'] is Map
            ? Map<String, dynamic>.from(json['status'] as Map)
            : json;

    return Device(
      id: _stringValue(json['device_id'] ?? json['id']),
      name: json['name'] ?? '',
      serialNumber: json['device_serial'] ?? json['serial_number'] ?? '',
      isOnline:
          json['is_online'] ??
          (lastSeen != null &&
              DateTime.now().difference(lastSeen).inMinutes < 5),
      lastSeen: lastSeen,
      status: DeviceStatus.fromJson(Map<String, dynamic>.from(statusJson)),
      config: DeviceConfig.fromJson(
        json['config'] is Map
            ? Map<String, dynamic>.from(json['config'] as Map)
            : const <String, dynamic>{},
      ),
    );
  }

  Map<String, dynamic> toJson() => {
    'id': id,
    'name': name,
    'serial_number': serialNumber,
    'is_online': isOnline,
    'last_seen': lastSeen?.toIso8601String(),
    'status': status.toJson(),
    'config': config.toJson(),
  };

  Device copyWith({
    String? id,
    String? name,
    String? serialNumber,
    bool? isOnline,
    DateTime? lastSeen,
    DeviceStatus? status,
    DeviceConfig? config,
  }) {
    return Device(
      id: id ?? this.id,
      name: name ?? this.name,
      serialNumber: serialNumber ?? this.serialNumber,
      isOnline: isOnline ?? this.isOnline,
      lastSeen: lastSeen ?? this.lastSeen,
      status: status ?? this.status,
      config: config ?? this.config,
    );
  }

  @override
  List<Object?> get props => [
    id,
    name,
    serialNumber,
    isOnline,
    lastSeen,
    status,
    config,
  ];
}

class DeviceStatus extends Equatable {
  final double batteryLevel;
  final double feedLevel;
  final double waterTemperature;
  final int signalStrength;
  final bool isSolarCharging;
  final double solarVoltage;
  final String connectionType;

  const DeviceStatus({
    required this.batteryLevel,
    required this.feedLevel,
    required this.waterTemperature,
    required this.signalStrength,
    required this.isSolarCharging,
    required this.solarVoltage,
    required this.connectionType,
  });

  factory DeviceStatus.fromJson(Map<String, dynamic> json) {
    return DeviceStatus(
      batteryLevel: (json['battery_level'] ?? 0).toDouble(),
      feedLevel:
          (json['feed_level'] ?? json['weight_percentage'] ?? 0).toDouble(),
      waterTemperature: (json['water_temperature'] ?? 0).toDouble(),
      signalStrength: json['signal_strength'] ?? json['cellular_signal'] ?? 0,
      isSolarCharging:
          json['is_solar_charging'] ??
          ((json['power_source'] ?? '').toString().toLowerCase() == 'solar'),
      solarVoltage: (json['solar_voltage'] ?? 0).toDouble(),
      connectionType:
          json['connection_type'] ?? json['power_source'] ?? 'unknown',
    );
  }

  Map<String, dynamic> toJson() => {
    'battery_level': batteryLevel,
    'feed_level': feedLevel,
    'water_temperature': waterTemperature,
    'signal_strength': signalStrength,
    'is_solar_charging': isSolarCharging,
    'solar_voltage': solarVoltage,
    'connection_type': connectionType,
  };

  @override
  List<Object?> get props => [
    batteryLevel,
    feedLevel,
    waterTemperature,
    signalStrength,
    isSolarCharging,
    solarVoltage,
    connectionType,
  ];
}

class DeviceConfig extends Equatable {
  final String timezone;
  final bool notificationsEnabled;
  final double lowFeedThreshold;
  final double lowBatteryThreshold;
  final double highTempThreshold;
  final double lowTempThreshold;

  const DeviceConfig({
    required this.timezone,
    required this.notificationsEnabled,
    required this.lowFeedThreshold,
    required this.lowBatteryThreshold,
    required this.highTempThreshold,
    required this.lowTempThreshold,
  });

  factory DeviceConfig.fromJson(Map<String, dynamic> json) {
    return DeviceConfig(
      timezone: json['timezone'] ?? 'UTC',
      notificationsEnabled: json['notifications_enabled'] ?? true,
      lowFeedThreshold: (json['low_feed_threshold'] ?? 20).toDouble(),
      lowBatteryThreshold: (json['low_battery_threshold'] ?? 20).toDouble(),
      highTempThreshold: (json['high_temp_threshold'] ?? 32).toDouble(),
      lowTempThreshold: (json['low_temp_threshold'] ?? 20).toDouble(),
    );
  }

  Map<String, dynamic> toJson() => {
    'timezone': timezone,
    'notifications_enabled': notificationsEnabled,
    'low_feed_threshold': lowFeedThreshold,
    'low_battery_threshold': lowBatteryThreshold,
    'high_temp_threshold': highTempThreshold,
    'low_temp_threshold': lowTempThreshold,
  };

  @override
  List<Object?> get props => [
    timezone,
    notificationsEnabled,
    lowFeedThreshold,
    lowBatteryThreshold,
    highTempThreshold,
    lowTempThreshold,
  ];
}
