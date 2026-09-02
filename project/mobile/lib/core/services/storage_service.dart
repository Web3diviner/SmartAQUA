import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../config/env_config.dart';

class StorageService {
  static late SharedPreferences _prefs;
  static const _secureStorage = FlutterSecureStorage(
    aOptions: AndroidOptions(),
    iOptions: IOSOptions(accessibility: KeychainAccessibility.first_unlock),
  );

  // Keys
  static const String _accessTokenKey = 'access_token';
  static const String _refreshTokenKey = 'refresh_token';
  static const String _userIdKey = 'user_id';
  static const String _deviceIdKey = 'device_id';
  static const String _mqttHostKey = 'mqtt_host';
  static const String _mqttPortKey = 'mqtt_port';
  static const String _biometricEnabledKey = 'biometric_enabled';
  static const String _themeKey = 'theme_mode';
  static const String _notificationsEnabledKey = 'notifications_enabled';
  static const String _alertNotificationsEnabledKey =
      'alert_notifications_enabled';
  static const String _feedingRemindersEnabledKey = 'feeding_reminders_enabled';
  static const String _temperatureUnitKey = 'temperature_unit';
  static const String _weightUnitKey = 'weight_unit';
  static const String _onboardingCompleteKey = 'onboarding_complete';

  static Future<void> init() async {
    _prefs = await SharedPreferences.getInstance();
  }

  // Secure Storage (for sensitive data)
  static Future<void> setAccessToken(String token) async {
    await _secureStorage.write(key: _accessTokenKey, value: token);
  }

  static Future<String?> getAccessToken() async {
    return await _secureStorage.read(key: _accessTokenKey);
  }

  static Future<void> setRefreshToken(String token) async {
    await _secureStorage.write(key: _refreshTokenKey, value: token);
  }

  static Future<String?> getRefreshToken() async {
    return await _secureStorage.read(key: _refreshTokenKey);
  }

  static Future<void> clearTokens() async {
    await _secureStorage.delete(key: _accessTokenKey);
    await _secureStorage.delete(key: _refreshTokenKey);
  }

  static Future<void> clearAll() async {
    await _secureStorage.deleteAll();
    await _prefs.clear();
  }

  // Shared Preferences (for non-sensitive data)
  static Future<void> setUserId(String userId) async {
    await _prefs.setString(_userIdKey, userId);
  }

  static String? getUserId() {
    return _prefs.getString(_userIdKey);
  }

  static Future<void> clearUserId() async {
    await _prefs.remove(_userIdKey);
  }

  static Future<void> setDeviceId(String deviceId) async {
    await _prefs.setString(_deviceIdKey, deviceId);
  }

  static String? getDeviceId() {
    return _prefs.getString(_deviceIdKey);
  }

  static Future<void> clearDeviceId() async {
    await _prefs.remove(_deviceIdKey);
  }

  static Future<void> setMqttHost(String host) async {
    await _prefs.setString(_mqttHostKey, host);
  }

  static String getMqttHost() {
    return _prefs.getString(_mqttHostKey) ?? EnvConfig.mqttHost;
  }

  static Future<void> setMqttPort(int port) async {
    await _prefs.setInt(_mqttPortKey, port);
  }

  static int getMqttPort() {
    return _prefs.getInt(_mqttPortKey) ?? EnvConfig.mqttPort;
  }

  static Future<void> setBiometricEnabled(bool enabled) async {
    await _prefs.setBool(_biometricEnabledKey, enabled);
  }

  static bool getBiometricEnabled() {
    return _prefs.getBool(_biometricEnabledKey) ?? false;
  }

  static Future<void> setThemeMode(String mode) async {
    await _prefs.setString(_themeKey, mode);
  }

  static String getThemeMode() {
    return _prefs.getString(_themeKey) ?? 'system';
  }

  static Future<void> setNotificationsEnabled(bool enabled) async {
    await _prefs.setBool(_notificationsEnabledKey, enabled);
  }

  static bool getNotificationsEnabled() {
    return _prefs.getBool(_notificationsEnabledKey) ?? true;
  }

  static Future<void> setAlertNotificationsEnabled(bool enabled) async {
    await _prefs.setBool(_alertNotificationsEnabledKey, enabled);
  }

  static bool getAlertNotificationsEnabled() {
    return _prefs.getBool(_alertNotificationsEnabledKey) ?? true;
  }

  static Future<void> setFeedingRemindersEnabled(bool enabled) async {
    await _prefs.setBool(_feedingRemindersEnabledKey, enabled);
  }

  static bool getFeedingRemindersEnabled() {
    return _prefs.getBool(_feedingRemindersEnabledKey) ?? true;
  }

  static Future<void> setTemperatureUnit(String unit) async {
    await _prefs.setString(_temperatureUnitKey, unit);
  }

  static String getTemperatureUnit() {
    return _prefs.getString(_temperatureUnitKey) ?? 'celsius';
  }

  static Future<void> setWeightUnit(String unit) async {
    await _prefs.setString(_weightUnitKey, unit);
  }

  static String getWeightUnit() {
    return _prefs.getString(_weightUnitKey) ?? 'grams';
  }

  // Onboarding
  static Future<void> setOnboardingComplete(bool complete) async {
    await _prefs.setBool(_onboardingCompleteKey, complete);
  }

  static bool isOnboardingComplete() {
    return _prefs.getBool(_onboardingCompleteKey) ?? false;
  }
}
