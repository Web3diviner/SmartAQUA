import 'package:flutter/material.dart';

class AppTheme {
  static const _primaryColor = Color(0xFF31B8B5);
  static const _secondaryColor = Color(0xFF3D66C6);
  static const _surfaceTint = Color(0xFFA9DCD7);
  static const _goldAccent = Color(0xFFC79A73);
  static const _errorColor = Color(0xFFE53935);

  static ThemeData get lightTheme {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.light,
      colorScheme: ColorScheme.fromSeed(
        seedColor: _primaryColor,
        secondary: _secondaryColor,
        surfaceTint: _surfaceTint,
        tertiary: _goldAccent,
        error: _errorColor,
        brightness: Brightness.light,
      ),
      appBarTheme: const AppBarTheme(
        centerTitle: true,
        elevation: 0,
        scrolledUnderElevation: 1,
      ),
      cardTheme: CardThemeData(
        elevation: 2,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        border: OutlineInputBorder(borderRadius: BorderRadius.circular(8)),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 12,
        ),
      ),
      snackBarTheme: SnackBarThemeData(
        behavior: SnackBarBehavior.floating,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      ),
    );
  }

  static ThemeData get darkTheme {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.dark,
      colorScheme: ColorScheme.fromSeed(
        seedColor: _primaryColor,
        secondary: _secondaryColor,
        surfaceTint: _surfaceTint,
        tertiary: _goldAccent,
        error: _errorColor,
        brightness: Brightness.dark,
      ),
      appBarTheme: const AppBarTheme(
        centerTitle: true,
        elevation: 0,
        scrolledUnderElevation: 1,
      ),
      cardTheme: CardThemeData(
        elevation: 2,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        border: OutlineInputBorder(borderRadius: BorderRadius.circular(8)),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 12,
        ),
      ),
      snackBarTheme: SnackBarThemeData(
        behavior: SnackBarBehavior.floating,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      ),
    );
  }

  // Custom colors for app-specific use
  static const Color feedLevelLow = Color(0xFFE53935);
  static const Color feedLevelMedium = Color(0xFFFF9800);
  static const Color feedLevelHigh = Color(0xFF4CAF50);
  static const Color batteryLow = Color(0xFFE53935);
  static const Color batteryMedium = Color(0xFFFF9800);
  static const Color batteryFull = Color(0xFF4CAF50);
  static const Color solarActive = Color(0xFFFFEB3B);
  static const Color deviceOnline = Color(0xFF4CAF50);
  static const Color deviceOffline = Color(0xFF9E9E9E);
  static const Color waterTempNormal = Color(0xFF2196F3);
  static const Color waterTempHigh = Color(0xFFE53935);
  static const Color waterTempLow = Color(0xFF03A9F4);
}
