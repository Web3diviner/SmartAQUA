import 'package:flutter/material.dart';

/// App color palette optimized for aquaculture IoT application
/// Colors chosen based on:
/// - Water/aquatic theme (blues, teals)
/// - Growth and health (greens)
/// - Alerts and warnings (amber, red)
/// - Professional and modern aesthetic
class AppColors {
  AppColors._();

  // Primary Colors - SmartAqua logo palette
  static const Color primary = Color(0xFF31B8B5); // Aqua teal
  static const Color primaryLight = Color(0xFF63CAC8); // Light aqua
  static const Color primaryDark = Color(0xFF1C8F8C); // Deep aqua
  static const Color primaryContainer = Color(0xFFA9DCD7); // Mint aqua

  // Secondary Colors - Ocean blue accent
  static const Color secondary = Color(0xFF3D66C6); // Ocean blue
  static const Color secondaryLight = Color(0xFF6A8DE0); // Light ocean blue
  static const Color secondaryDark = Color(0xFF294CA5); // Deep ocean blue
  static const Color secondaryContainer = Color(
    0xFFC7D6F5,
  ); // Very light ocean blue

  // Tertiary Colors - Circuit gold accent
  static const Color tertiary = Color(0xFFC79A73); // Gold accent
  static const Color tertiaryLight = Color(0xFFD9B594); // Light gold
  static const Color tertiaryDark = Color(0xFFAA7C54); // Deep gold
  static const Color tertiaryContainer = Color(0xFFF0DFCF); // Very light gold

  // Semantic Colors - Status
  static const Color success = Color(0xFF43A047); // Green
  static const Color successLight = Color(0xFF76D275); // Light green
  static const Color successDark = Color(0xFF2E7D32); // Dark green
  static const Color successContainer = Color(0xFFC8E6C9); // Very light green

  static const Color warning = Color(0xFFFB8C00); // Orange
  static const Color warningLight = Color(0xFFFFB74D); // Light orange
  static const Color warningDark = Color(0xFFE65100); // Dark orange
  static const Color warningContainer = Color(0xFFFFE0B2); // Very light orange

  static const Color error = Color(0xFFE53935); // Red
  static const Color errorLight = Color(0xFFEF5350); // Light red
  static const Color errorDark = Color(0xFFC62828); // Dark red
  static const Color errorContainer = Color(0xFFFFCDD2); // Very light red

  static const Color info = Color(0xFF1976D2); // Blue
  static const Color infoLight = Color(0xFF64B5F6); // Light blue
  static const Color infoDark = Color(0xFF0D47A1); // Dark blue
  static const Color infoContainer = Color(0xFFBBDEFB); // Very light blue

  // Neutral Colors
  static const Color background = Color(0xFFEAF7F6); // Soft mint white
  static const Color backgroundDark = Color(0xFF121212); // Dark background
  static const Color surface = Color(0xFFFFFFFF); // White
  static const Color surfaceDark = Color(0xFF1E1E1E); // Dark surface
  static const Color surfaceVariant = Color(0xFFF5F5F5); // Light gray
  static const Color surfaceVariantDark = Color(0xFF2C2C2C); // Dark gray

  // Text Colors
  static const Color onPrimary = Color(0xFFFFFFFF); // White on primary
  static const Color onSecondary = Color(0xFFFFFFFF); // White on secondary
  static const Color onTertiary = Color(0xFFFFFFFF); // White on tertiary
  static const Color onBackground = Color(
    0xFF212121,
  ); // Dark gray on background
  static const Color onBackgroundDark = Color(0xFFE0E0E0); // Light gray on dark
  static const Color onSurface = Color(0xFF212121); // Dark gray on surface
  static const Color onSurfaceDark = Color(
    0xFFE0E0E0,
  ); // Light gray on dark surface
  static const Color onError = Color(0xFFFFFFFF); // White on error

  // Text Hierarchy
  static const Color textPrimary = Color(0xFF212121); // Primary text
  static const Color textPrimaryDark = Color(0xFFE0E0E0); // Primary text dark
  static const Color textSecondary = Color(0xFF757575); // Secondary text
  static const Color textSecondaryDark = Color(
    0xFFB0B0B0,
  ); // Secondary text dark
  static const Color textDisabled = Color(0xFFBDBDBD); // Disabled text
  static const Color textDisabledDark = Color(0xFF616161); // Disabled text dark

  // Border & Divider
  static const Color border = Color(0xFFE0E0E0); // Light border
  static const Color borderDark = Color(0xFF424242); // Dark border
  static const Color divider = Color(0xFFBDBDBD); // Divider
  static const Color dividerDark = Color(0xFF616161); // Dark divider

  // Overlay & Shadow
  static const Color overlay = Color(0x1F000000); // 12% black
  static const Color overlayDark = Color(0x3D000000); // 24% black
  static const Color shadow = Color(0x1A000000); // 10% black
  static const Color shadowDark = Color(0x52000000); // 32% black

  // App-Specific Semantic Colors

  // Device Status
  static const Color deviceOnline = Color(0xFF43A047); // Green
  static const Color deviceOffline = Color(0xFF9E9E9E); // Gray
  static const Color deviceError = Color(0xFFE53935); // Red
  static const Color deviceWarning = Color(0xFFFB8C00); // Orange

  // Battery Levels
  static const Color batteryFull = Color(0xFF43A047); // Green
  static const Color batteryMedium = Color(0xFFFB8C00); // Orange
  static const Color batteryLow = Color(0xFFE53935); // Red
  static const Color batteryCharging = Color(0xFF1976D2); // Blue

  // Feed Levels
  static const Color feedHigh = Color(0xFF43A047); // Green
  static const Color feedMedium = Color(0xFFFB8C00); // Orange
  static const Color feedLow = Color(0xFFE53935); // Red
  static const Color feedEmpty = Color(0xFF9E9E9E); // Gray

  // Water Quality
  static const Color waterOptimal = Color(0xFF00897B); // Teal
  static const Color waterGood = Color(0xFF43A047); // Green
  static const Color waterWarning = Color(0xFFFB8C00); // Orange
  static const Color waterCritical = Color(0xFFE53935); // Red

  // Temperature
  static const Color tempNormal = Color(0xFF0277BD); // Blue
  static const Color tempHigh = Color(0xFFE53935); // Red
  static const Color tempLow = Color(0xFF1976D2); // Light blue

  // Dissolved Oxygen
  static const Color oxygenHigh = Color(0xFF43A047); // Green
  static const Color oxygenNormal = Color(0xFF00897B); // Teal
  static const Color oxygenLow = Color(0xFFFB8C00); // Orange
  static const Color oxygenCritical = Color(0xFFE53935); // Red

  // Solar/Power
  static const Color solarActive = Color(0xFFFFB300); // Yellow
  static const Color solarInactive = Color(0xFF9E9E9E); // Gray
  static const Color powerAC = Color(0xFF1976D2); // Blue
  static const Color powerBattery = Color(0xFF43A047); // Green

  // Connectivity
  static const Color wifiStrong = Color(0xFF43A047); // Green
  static const Color wifiMedium = Color(0xFFFB8C00); // Orange
  static const Color wifiWeak = Color(0xFFE53935); // Red
  static const Color cellularStrong = Color(0xFF00897B); // Teal
  static const Color cellularMedium = Color(0xFFFB8C00); // Orange
  static const Color cellularWeak = Color(0xFFE53935); // Red
  static const Color bluetoothConnected = Color(0xFF1976D2); // Blue
  static const Color bluetoothDisconnected = Color(0xFF9E9E9E); // Gray

  // Chart Colors (for data visualization)
  static const List<Color> chartColors = [
    Color(0xFF31B8B5), // Aqua teal
    Color(0xFF3D66C6), // Ocean blue
    Color(0xFFC79A73), // Gold accent
    Color(0xFF43A047), // Green
    Color(0xFFFB8C00), // Orange
    Color(0xFFE53935), // Red
    Color(0xFF00ACC1), // Cyan
    Color(0xFF7CB342), // Light green
  ];

  // Gradient Definitions
  static const LinearGradient primaryGradient = LinearGradient(
    colors: [primary, primaryLight],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  static const LinearGradient secondaryGradient = LinearGradient(
    colors: [secondary, secondaryLight],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  static const LinearGradient successGradient = LinearGradient(
    colors: [success, successLight],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  static const LinearGradient waterGradient = LinearGradient(
    colors: [Color(0xFF3D66C6), Color(0xFF31B8B5)],
    begin: Alignment.topCenter,
    end: Alignment.bottomCenter,
  );

  static const LinearGradient sunsetGradient = LinearGradient(
    colors: [Color(0xFFFF6F00), Color(0xFFFFB300), Color(0xFFFFD54F)],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  // Helper method to get color with opacity
  static Color withOpacity(Color color, double opacity) {
    return color.withValues(alpha: opacity);
  }

  // Helper method to get battery color based on level
  static Color getBatteryColor(double level) {
    if (level > 50) return batteryFull;
    if (level > 20) return batteryMedium;
    return batteryLow;
  }

  // Helper method to get feed level color
  static Color getFeedLevelColor(double level) {
    if (level > 50) return feedHigh;
    if (level > 20) return feedMedium;
    if (level > 0) return feedLow;
    return feedEmpty;
  }

  // Helper method to get water quality color
  static Color getWaterQualityColor(double quality) {
    if (quality >= 90) return waterOptimal;
    if (quality >= 70) return waterGood;
    if (quality >= 50) return waterWarning;
    return waterCritical;
  }

  // Helper method to get signal strength color
  static Color getSignalStrengthColor(int strength) {
    if (strength >= 75) return wifiStrong;
    if (strength >= 50) return wifiMedium;
    return wifiWeak;
  }
}
