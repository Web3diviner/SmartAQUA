import 'package:equatable/equatable.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../services/storage_service.dart';

class AppPreferencesState extends Equatable {
  final ThemeMode themeMode;
  final bool notificationsEnabled;
  final bool alertNotificationsEnabled;
  final bool feedingRemindersEnabled;
  final String temperatureUnit;
  final String weightUnit;

  const AppPreferencesState({
    required this.themeMode,
    required this.notificationsEnabled,
    required this.alertNotificationsEnabled,
    required this.feedingRemindersEnabled,
    required this.temperatureUnit,
    required this.weightUnit,
  });

  factory AppPreferencesState.fromStorage() {
    return AppPreferencesState(
      themeMode: _themeModeFromStorage(StorageService.getThemeMode()),
      notificationsEnabled: StorageService.getNotificationsEnabled(),
      alertNotificationsEnabled: StorageService.getAlertNotificationsEnabled(),
      feedingRemindersEnabled: StorageService.getFeedingRemindersEnabled(),
      temperatureUnit: StorageService.getTemperatureUnit(),
      weightUnit: StorageService.getWeightUnit(),
    );
  }

  AppPreferencesState copyWith({
    ThemeMode? themeMode,
    bool? notificationsEnabled,
    bool? alertNotificationsEnabled,
    bool? feedingRemindersEnabled,
    String? temperatureUnit,
    String? weightUnit,
  }) {
    return AppPreferencesState(
      themeMode: themeMode ?? this.themeMode,
      notificationsEnabled: notificationsEnabled ?? this.notificationsEnabled,
      alertNotificationsEnabled:
          alertNotificationsEnabled ?? this.alertNotificationsEnabled,
      feedingRemindersEnabled:
          feedingRemindersEnabled ?? this.feedingRemindersEnabled,
      temperatureUnit: temperatureUnit ?? this.temperatureUnit,
      weightUnit: weightUnit ?? this.weightUnit,
    );
  }

  @override
  List<Object?> get props => [
    themeMode,
    notificationsEnabled,
    alertNotificationsEnabled,
    feedingRemindersEnabled,
    temperatureUnit,
    weightUnit,
  ];

  static ThemeMode _themeModeFromStorage(String storedValue) {
    switch (storedValue) {
      case 'light':
        return ThemeMode.light;
      case 'dark':
        return ThemeMode.dark;
      default:
        return ThemeMode.system;
    }
  }
}

class AppPreferencesNotifier extends StateNotifier<AppPreferencesState> {
  AppPreferencesNotifier() : super(AppPreferencesState.fromStorage());

  Future<void> setThemeMode(ThemeMode themeMode) async {
    await StorageService.setThemeMode(_themeModeToStorage(themeMode));
    state = state.copyWith(themeMode: themeMode);
  }

  Future<void> setNotificationsEnabled(bool enabled) async {
    await StorageService.setNotificationsEnabled(enabled);
    state = state.copyWith(notificationsEnabled: enabled);
  }

  Future<void> setAlertNotificationsEnabled(bool enabled) async {
    await StorageService.setAlertNotificationsEnabled(enabled);
    state = state.copyWith(alertNotificationsEnabled: enabled);
  }

  Future<void> setFeedingRemindersEnabled(bool enabled) async {
    await StorageService.setFeedingRemindersEnabled(enabled);
    state = state.copyWith(feedingRemindersEnabled: enabled);
  }

  Future<void> setTemperatureUnit(String unit) async {
    await StorageService.setTemperatureUnit(unit);
    state = state.copyWith(temperatureUnit: unit);
  }

  Future<void> setWeightUnit(String unit) async {
    await StorageService.setWeightUnit(unit);
    state = state.copyWith(weightUnit: unit);
  }

  String _themeModeToStorage(ThemeMode themeMode) {
    switch (themeMode) {
      case ThemeMode.light:
        return 'light';
      case ThemeMode.dark:
        return 'dark';
      case ThemeMode.system:
        return 'system';
    }
  }
}

final appPreferencesProvider =
    StateNotifierProvider<AppPreferencesNotifier, AppPreferencesState>((ref) {
      return AppPreferencesNotifier();
    });
