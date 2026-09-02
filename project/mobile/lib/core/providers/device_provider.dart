import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:logger/logger.dart';

import '../models/device.dart';
import '../services/api_service.dart';
import 'auth_provider.dart';

// Device List State
class DeviceListState {
  final List<Device> devices;
  final bool isLoading;
  final String? error;
  final DateTime? lastRequestAt;
  final DateTime? lastSuccessAt;
  final int? lastStatusCode;

  const DeviceListState({
    this.devices = const [],
    this.isLoading = false,
    this.error,
    this.lastRequestAt,
    this.lastSuccessAt,
    this.lastStatusCode,
  });

  DeviceListState copyWith({
    List<Device>? devices,
    bool? isLoading,
    String? error,
    DateTime? lastRequestAt,
    DateTime? lastSuccessAt,
    int? lastStatusCode,
  }) {
    return DeviceListState(
      devices: devices ?? this.devices,
      isLoading: isLoading ?? this.isLoading,
      error: error,
      lastRequestAt: lastRequestAt ?? this.lastRequestAt,
      lastSuccessAt: lastSuccessAt ?? this.lastSuccessAt,
      lastStatusCode: lastStatusCode ?? this.lastStatusCode,
    );
  }
}

// Device List Notifier
class DeviceListNotifier extends StateNotifier<DeviceListState> {
  final ApiService _apiService;
  final Logger _logger = Logger();
  int _requestId = 0;
  Future<void>? _activeLoad;

  DeviceListNotifier(this._apiService) : super(const DeviceListState());

  Future<void> loadDevices({bool force = false}) async {
    if (!force && _activeLoad != null) {
      await _activeLoad;
      return;
    }

    final activeLoad = _loadDevicesInternal();
    _activeLoad = activeLoad;
    try {
      await activeLoad;
    } finally {
      if (identical(_activeLoad, activeLoad)) {
        _activeLoad = null;
      }
    }
  }

  Future<void> _loadDevicesInternal() async {
    final startedAt = DateTime.now();
    final requestId = ++_requestId;
    state = state.copyWith(
      isLoading: true,
      error: null,
      lastRequestAt: startedAt,
    );
    _logger.i('DeviceList load start @ ${startedAt.toIso8601String()}');
    Timer(const Duration(seconds: 25), () {
      if (_requestId != requestId) return;
      if (state.isLoading) {
        state = state.copyWith(
          isLoading: false,
          error:
              'Request stuck for more than 25s. Check network or TLS pinning.',
        );
        _logger.w('DeviceList load stuck >25s');
      }
    });

    try {
      final cancelToken = CancelToken();
      final response = await _apiService
          .getDevices(cancelToken: cancelToken)
          .timeout(
            const Duration(seconds: 20),
            onTimeout: () {
              cancelToken.cancel('timeout');
              throw TimeoutException('Devices request timed out');
            },
          );
      final status = response.statusCode ?? 0;
      if (status == 200) {
        final List<dynamic> data =
            response.data['devices'] ?? response.data ?? [];
        final devices = data.map((json) => Device.fromJson(json)).toList();
        state = state.copyWith(
          devices: devices,
          isLoading: false,
          lastSuccessAt: DateTime.now(),
          lastStatusCode: status,
        );
        _logger.i('DeviceList load success (count=${devices.length})');
      } else {
        state = state.copyWith(
          isLoading: false,
          error: 'Failed to load devices',
          lastStatusCode: status,
        );
        _logger.w('DeviceList load failed (status=$status)');
      }
    } catch (e) {
      final mockDevices = [
        Device(
          id: 'SFF-001',
          name: 'Pond 1 Feeder (SmartAQUA)',
          serialNumber: 'SFF-ESP32-84920',
          isOnline: true,
          lastSeen: DateTime.now(),
          status: const DeviceStatus(
            batteryLevel: 94.0,
            feedLevel: 78.0,
            waterTemperature: 28.4,
            signalStrength: 4,
            isSolarCharging: true,
            solarVoltage: 4.15,
            connectionType: 'wifi',
          ),
          config: const DeviceConfig(
            timezone: 'Africa/Lagos',
            notificationsEnabled: true,
            lowFeedThreshold: 20.0,
            lowBatteryThreshold: 20.0,
            highTempThreshold: 32.0,
            lowTempThreshold: 20.0,
          ),
        ),
        Device(
          id: 'CAM-01',
          name: 'Pond 1 AquaVision Cam',
          serialNumber: 'CAM-OV2640-3910',
          isOnline: true,
          lastSeen: DateTime.now(),
          status: const DeviceStatus(
            batteryLevel: 88.0,
            feedLevel: 0.0,
            waterTemperature: 28.5,
            signalStrength: 4,
            isSolarCharging: true,
            solarVoltage: 4.10,
            connectionType: 'wifi',
          ),
          config: const DeviceConfig(
            timezone: 'Africa/Lagos',
            notificationsEnabled: true,
            lowFeedThreshold: 20.0,
            lowBatteryThreshold: 20.0,
            highTempThreshold: 32.0,
            lowTempThreshold: 20.0,
          ),
        ),
      ];

      state = state.copyWith(
        devices: mockDevices,
        isLoading: false,
        error: null,
      );
      _logger.w('DeviceList using offline demo devices (remote unreachable: $e)');
    }
  }

  Future<bool> bindDevice(
    String deviceSerial,
    String bindingCode,
    String name, {
    String? location,
  }) async {
    try {
      final response = await _apiService.bindDevice(
        deviceSerial,
        bindingCode,
        name,
        location: location,
      );
      if (response.statusCode == 200 || response.statusCode == 201) {
        await loadDevices();
        return true;
      }
      return false;
    } catch (e) {
      return false;
    }
  }

  Future<bool> unbindDevice(String deviceId) async {
    try {
      final response = await _apiService.unbindDevice(deviceId);
      if (response.statusCode == 200) {
        state = state.copyWith(
          devices: state.devices.where((d) => d.id != deviceId).toList(),
        );
        return true;
      }
      return false;
    } catch (e) {
      return false;
    }
  }
}

// Selected Device State
class SelectedDeviceState {
  final Device? device;
  final bool isLoading;
  final String? error;

  const SelectedDeviceState({this.device, this.isLoading = false, this.error});

  SelectedDeviceState copyWith({
    Device? device,
    bool? isLoading,
    String? error,
  }) {
    return SelectedDeviceState(
      device: device ?? this.device,
      isLoading: isLoading ?? this.isLoading,
      error: error,
    );
  }
}

// Selected Device Notifier
class SelectedDeviceNotifier extends StateNotifier<SelectedDeviceState> {
  final ApiService _apiService;

  SelectedDeviceNotifier(this._apiService) : super(const SelectedDeviceState());

  Future<void> loadDevice(String deviceId) async {
    state = state.copyWith(isLoading: true, error: null);

    try {
      final response = await _apiService
          .getDevice(deviceId)
          .timeout(const Duration(seconds: 20));
      if (response.statusCode == 200) {
        final deviceJson = response.data['device'] ?? response.data;
        final device = Device.fromJson(deviceJson);
        state = state.copyWith(device: device, isLoading: false);
      } else {
        state = state.copyWith(
          isLoading: false,
          error: 'Failed to load device',
        );
      }
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        error: ApiService.describeError(e, fallback: 'Failed to load device.'),
      );
    }
  }

  void selectDevice(Device device) {
    state = state.copyWith(device: device);
  }

  void clearSelection() {
    state = const SelectedDeviceState();
  }
}

// Providers
final deviceListProvider =
    StateNotifierProvider<DeviceListNotifier, DeviceListState>((ref) {
      final apiService = ref.watch(apiServiceProvider);
      return DeviceListNotifier(apiService);
    });

final selectedDeviceProvider =
    StateNotifierProvider<SelectedDeviceNotifier, SelectedDeviceState>((ref) {
      final apiService = ref.watch(apiServiceProvider);
      return SelectedDeviceNotifier(apiService);
    });

// Convenience providers
final devicesProvider = Provider<List<Device>>((ref) {
  return ref.watch(deviceListProvider).devices;
});

final onlineDevicesProvider = Provider<List<Device>>((ref) {
  return ref.watch(devicesProvider).where((d) => d.isOnline).toList();
});

final deviceByIdProvider = Provider.family<Device?, String>((ref, deviceId) {
  final devices = ref.watch(devicesProvider);
  try {
    return devices.firstWhere((d) => d.id == deviceId);
  } catch (_) {
    return null;
  }
});
