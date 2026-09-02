import 'package:flutter/material.dart';

class AppTheme {
  // Brand Color Palette - Deep Sea Obsidian & Bioluminescent Cyan
  static const primaryCyan = Color(0xFF00E5FF);
  static const primaryTeal = Color(0xFF00E5A3);
  static const deepSeaCobalt = Color(0xFF0077B6);
  static const abyssalNavy = Color(0xFF070F18);
  static const darkCardBg = Color(0xFF0D1B2A);
  static const darkSurfaceBg = Color(0xFF142436);
  static const neonPurple = Color(0xFF7928CA);
  static const neonPink = Color(0xFFFF0080);
  static const neonAmber = Color(0xFFFFB703);

  // Status Colors
  static const Color feedLevelLow = Color(0xFFFF4D4D);
  static const Color feedLevelMedium = Color(0xFFFFB703);
  static const Color feedLevelHigh = Color(0xFF00E5A3);
  static const Color batteryLow = Color(0xFFFF4D4D);
  static const Color batteryMedium = Color(0xFFFFB703);
  static const Color batteryFull = Color(0xFF00E5A3);
  static const Color solarActive = Color(0xFFFFD166);
  static const Color deviceOnline = Color(0xFF00E5A3);
  static const Color deviceOffline = Color(0xFF6C7A89);
  static const Color waterTempNormal = Color(0xFF00E5FF);
  static const Color waterTempHigh = Color(0xFFFF4D4D);
  static const Color waterTempLow = Color(0xFF00B4D8);

  // Custom Gradients
  static const LinearGradient primaryGradient = LinearGradient(
    colors: [Color(0xFF00E5FF), Color(0xFF0077B6)],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  static const LinearGradient aquaDocGradient = LinearGradient(
    colors: [Color(0xFF7928CA), Color(0xFF00E5FF)],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  static const LinearGradient successGradient = LinearGradient(
    colors: [Color(0xFF00E5A3), Color(0xFF00A86B)],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  static const LinearGradient cardDarkGradient = LinearGradient(
    colors: [Color(0xFF0E1E2E), Color(0xFF08131D)],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  static const LinearGradient glassOverlayGradient = LinearGradient(
    colors: [Color(0x2000E5FF), Color(0x050077B6)],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  static ThemeData get lightTheme {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.light,
      scaffoldBackgroundColor: const Color(0xFFF4F7FB),
      colorScheme: ColorScheme.fromSeed(
        seedColor: primaryCyan,
        primary: const Color(0xFF00838F),
        secondary: deepSeaCobalt,
        surface: Colors.white,
        error: const Color(0xFFD32F2F),
        brightness: Brightness.light,
      ),
      fontFamily: 'Inter',
      appBarTheme: const AppBarTheme(
        centerTitle: true,
        elevation: 0,
        backgroundColor: Colors.white,
        foregroundColor: Color(0xFF0B192C),
        titleTextStyle: TextStyle(
          color: Color(0xFF0B192C),
          fontSize: 18,
          fontWeight: FontWeight.w700,
          letterSpacing: -0.3,
        ),
      ),
      cardTheme: CardThemeData(
        elevation: 0,
        color: Colors.white,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
          side: const BorderSide(color: Color(0xFFE2E8F0), width: 1),
        ),
      ),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: Colors.white,
        indicatorColor: const Color(0xFF00838F).withOpacity(0.15),
        labelTextStyle: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return const TextStyle(fontSize: 12, fontWeight: FontWeight.bold, color: Color(0xFF00838F));
          }
          return const TextStyle(fontSize: 12, fontWeight: FontWeight.w500, color: Color(0xFF64748B));
        }),
        iconTheme: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return const IconThemeData(color: Color(0xFF00838F), size: 24);
          }
          return const IconThemeData(color: Color(0xFF64748B), size: 24);
        }),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          elevation: 2,
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
          backgroundColor: const Color(0xFF00838F),
          foregroundColor: Colors.white,
          textStyle: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: const Color(0xFFF8FAFC),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: Color(0xFFCBD5E1)),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: Color(0xFFE2E8F0)),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: Color(0xFF00838F), width: 1.5),
        ),
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      ),
    );
  }

  static ThemeData get darkTheme {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.dark,
      scaffoldBackgroundColor: abyssalNavy,
      colorScheme: ColorScheme.fromSeed(
        seedColor: primaryCyan,
        primary: primaryCyan,
        secondary: primaryTeal,
        surface: darkCardBg,
        error: feedLevelLow,
        brightness: Brightness.dark,
      ),
      fontFamily: 'Inter',
      appBarTheme: const AppBarTheme(
        centerTitle: true,
        elevation: 0,
        backgroundColor: abyssalNavy,
        foregroundColor: Colors.white,
        titleTextStyle: TextStyle(
          color: Colors.white,
          fontSize: 18,
          fontWeight: FontWeight.w700,
          letterSpacing: -0.3,
        ),
      ),
      cardTheme: CardThemeData(
        elevation: 0,
        color: darkCardBg,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(18),
          side: BorderSide(color: Colors.white.withOpacity(0.08), width: 1),
        ),
      ),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: const Color(0xFF0A1522),
        indicatorColor: primaryCyan.withOpacity(0.18),
        elevation: 8,
        labelTextStyle: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return const TextStyle(fontSize: 12, fontWeight: FontWeight.bold, color: primaryCyan);
          }
          return const TextStyle(fontSize: 12, fontWeight: FontWeight.w500, color: Color(0xFF8899A6));
        }),
        iconTheme: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return const IconThemeData(color: primaryCyan, size: 24);
          }
          return const IconThemeData(color: Color(0xFF8899A6), size: 24);
        }),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          elevation: 4,
          shadowColor: primaryCyan.withOpacity(0.35),
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
          backgroundColor: primaryCyan,
          foregroundColor: const Color(0xFF04101A),
          textStyle: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: darkSurfaceBg,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(color: Colors.white.withOpacity(0.12)),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(color: Colors.white.withOpacity(0.08)),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: primaryCyan, width: 1.5),
        ),
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      ),
    );
  }
}
