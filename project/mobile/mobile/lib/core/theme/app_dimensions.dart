/// Spacing and dimension constants for consistent layout
class AppDimensions {
  AppDimensions._();

  // Spacing Scale (8px base unit)
  static const double space0 = 0;
  static const double space1 = 4; // 0.5x
  static const double space2 = 8; // 1x (base)
  static const double space3 = 12; // 1.5x
  static const double space4 = 16; // 2x
  static const double space5 = 20; // 2.5x
  static const double space6 = 24; // 3x
  static const double space7 = 28; // 3.5x
  static const double space8 = 32; // 4x
  static const double space10 = 40; // 5x
  static const double space12 = 48; // 6x
  static const double space16 = 64; // 8x
  static const double space20 = 80; // 10x
  static const double space24 = 96; // 12x

  // Semantic Spacing
  static const double paddingXS = space1; // 4
  static const double paddingSM = space2; // 8
  static const double paddingMD = space4; // 16
  static const double paddingLG = space6; // 24
  static const double paddingXL = space8; // 32
  static const double paddingXXL = space12; // 48

  static const double marginXS = space1; // 4
  static const double marginSM = space2; // 8
  static const double marginMD = space4; // 16
  static const double marginLG = space6; // 24
  static const double marginXL = space8; // 32
  static const double marginXXL = space12; // 48

  // Border Radius
  static const double radiusXS = 4;
  static const double radiusSM = 8;
  static const double radiusMD = 12;
  static const double radiusLG = 16;
  static const double radiusXL = 20;
  static const double radiusXXL = 24;
  static const double radiusFull = 9999; // Fully rounded

  // Border Width
  static const double borderThin = 1;
  static const double borderMedium = 2;
  static const double borderThick = 4;

  // Icon Sizes
  static const double iconXS = 16;
  static const double iconSM = 20;
  static const double iconMD = 24;
  static const double iconLG = 32;
  static const double iconXL = 48;
  static const double iconXXL = 64;

  // Button Heights
  static const double buttonHeightSM = 32;
  static const double buttonHeightMD = 40;
  static const double buttonHeightLG = 48;
  static const double buttonHeightXL = 56;

  // Input Heights
  static const double inputHeightSM = 36;
  static const double inputHeightMD = 44;
  static const double inputHeightLG = 52;

  // Card Dimensions
  static const double cardPadding = paddingMD;
  static const double cardRadius = radiusMD;
  static const double cardElevation = 2;

  // AppBar
  static const double appBarHeight = 56;
  static const double appBarElevation = 0;

  // Bottom Navigation
  static const double bottomNavHeight = 64;
  static const double bottomNavElevation = 8;

  // FAB
  static const double fabSize = 56;
  static const double fabSizeMini = 40;
  static const double fabElevation = 6;

  // Divider
  static const double dividerThickness = 1;
  static const double dividerIndent = paddingMD;

  // List Tile
  static const double listTileHeight = 56;
  static const double listTilePadding = paddingMD;

  // Avatar Sizes
  static const double avatarXS = 24;
  static const double avatarSM = 32;
  static const double avatarMD = 40;
  static const double avatarLG = 56;
  static const double avatarXL = 72;
  static const double avatarXXL = 96;

  // Chip
  static const double chipHeight = 32;
  static const double chipPadding = paddingSM;
  static const double chipRadius = radiusFull;

  // Dialog
  static const double dialogMaxWidth = 560;
  static const double dialogPadding = paddingLG;
  static const double dialogRadius = radiusLG;

  // Bottom Sheet
  static const double bottomSheetRadius = radiusLG;
  static const double bottomSheetMaxHeight = 0.9; // 90% of screen height

  // Snackbar
  static const double snackbarRadius = radiusSM;
  static const double snackbarPadding = paddingMD;

  // Chart
  static const double chartHeight = 200;
  static const double chartPadding = paddingMD;

  // Gauge
  static const double gaugeSizeSM = 80;
  static const double gaugeSizeMD = 120;
  static const double gaugeSizeLG = 160;

  // Stat Card
  static const double statCardHeight = 120;
  static const double statCardPadding = paddingMD;

  // Device Card
  static const double deviceCardHeight = 160;
  static const double deviceCardPadding = paddingMD;

  // Screen Padding
  static const double screenPaddingHorizontal = paddingMD;
  static const double screenPaddingVertical = paddingMD;

  // Grid
  static const double gridSpacing = paddingMD;
  static const int gridCrossAxisCount = 2;

  // Animation Durations (milliseconds)
  static const int animationFast = 150;
  static const int animationNormal = 300;
  static const int animationSlow = 500;

  // Breakpoints (for responsive design)
  static const double breakpointMobile = 600;
  static const double breakpointTablet = 900;
  static const double breakpointDesktop = 1200;

  // Max Content Width
  static const double maxContentWidth = 1200;

  // Elevation Levels
  static const double elevation0 = 0;
  static const double elevation1 = 1;
  static const double elevation2 = 2;
  static const double elevation3 = 3;
  static const double elevation4 = 4;
  static const double elevation6 = 6;
  static const double elevation8 = 8;
  static const double elevation12 = 12;
  static const double elevation16 = 16;
  static const double elevation24 = 24;
}
