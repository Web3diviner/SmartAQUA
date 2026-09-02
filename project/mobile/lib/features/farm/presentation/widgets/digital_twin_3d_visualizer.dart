import 'dart:math' as math;
import 'package:flutter/material.dart';

enum CameraPreset { isometric, topDown, surface, underwater }

class Fish3D {
  double x;
  double y; // depth (vertical in tank: 0 is surface, 1 is bottom)
  double z;
  double vx;
  double vy;
  double vz;
  double yaw;
  double length;
  double phase;
  double speed;
  Color color;

  Fish3D({
    required this.x,
    required this.y,
    required this.z,
    required this.vx,
    required this.vy,
    required this.vz,
    required this.yaw,
    required this.length,
    required this.phase,
    required this.speed,
    required this.color,
  });
}

class Bubble3D {
  double x;
  double y;
  double z;
  double speed;
  double size;
  double wobblePhase;

  Bubble3D({
    required this.x,
    required this.y,
    required this.z,
    required this.speed,
    required this.size,
    required this.wobblePhase,
  });
}

class DigitalTwin3DVisualizer extends StatefulWidget {
  final double dissolvedOxygen;
  final double temperature;
  final double ammoniaTan;
  final double avgWeightG;
  final double biomassKg;
  final int population;
  final String systemType; // 'concrete', 'tarpaulin', 'earthen', 'ras'
  final bool autoRotate;

  const DigitalTwin3DVisualizer({
    super.key,
    this.dissolvedOxygen = 5.8,
    this.temperature = 28.4,
    this.ammoniaTan = 0.15,
    this.avgWeightG = 320.0,
    this.biomassKg = 1552.0,
    this.population = 4850,
    this.systemType = 'concrete',
    this.autoRotate = true,
  });

  @override
  State<DigitalTwin3DVisualizer> createState() => _DigitalTwin3DVisualizerState();
}

class _DigitalTwin3DVisualizerState extends State<DigitalTwin3DVisualizer>
    with SingleTickerProviderStateMixin {
  late AnimationController _animController;
  final List<Fish3D> _fishes = [];
  final List<Bubble3D> _bubbles = [];
  final math.Random _rng = math.Random();

  double _rotX = 0.45; // pitch
  double _rotY = -0.65; // yaw
  double _zoom = 1.0;
  CameraPreset _preset = CameraPreset.isometric;
  bool _isAutoRotating = true;

  @override
  void initState() {
    super.initState();
    _isAutoRotating = widget.autoRotate;
    _initFishes();
    _initBubbles();

    _animController = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 10),
    )..addListener(_updatePhysics)..repeat();
  }

  @override
  void dispose() {
    _animController.dispose();
    super.dispose();
  }

  void _initFishes() {
    _fishes.clear();
    const fishCount = 28;
    for (int i = 0; i < fishCount; i++) {
      final yaw = _rng.nextDouble() * 2 * math.pi;
      final speed = 0.008 + _rng.nextDouble() * 0.008;
      _fishes.add(
        Fish3D(
          x: (_rng.nextDouble() - 0.5) * 1.4,
          y: 0.2 + _rng.nextDouble() * 0.6,
          z: (_rng.nextDouble() - 0.5) * 1.4,
          vx: math.cos(yaw) * speed,
          vy: (_rng.nextDouble() - 0.5) * 0.002,
          vz: math.sin(yaw) * speed,
          yaw: yaw,
          length: (0.18 + _rng.nextDouble() * 0.12) * _getWeightScale(),
          phase: _rng.nextDouble() * 2 * math.pi,
          speed: speed,
          color: i % 4 == 0
              ? const Color(0xFF2C3E50)
              : (i % 2 == 0 ? const Color(0xFF34495E) : const Color(0xFF1B2631)),
        ),
      );
    }
  }

  void _initBubbles() {
    _bubbles.clear();
    for (int i = 0; i < 40; i++) {
      _bubbles.add(
        Bubble3D(
          x: (_rng.nextDouble() - 0.5) * 0.8,
          y: 0.1 + _rng.nextDouble() * 0.9,
          z: (_rng.nextDouble() - 0.5) * 0.8,
          speed: 0.006 + _rng.nextDouble() * 0.008,
          size: 2.0 + _rng.nextDouble() * 3.5,
          wobblePhase: _rng.nextDouble() * 2 * math.pi,
        ),
      );
    }
  }

  double _getWeightScale() {
    final w = widget.avgWeightG;
    return (math.pow(w / 150.0, 0.33)).clamp(0.6, 1.8).toDouble();
  }

  void _updatePhysics() {
    if (!mounted) return;

    final isHypoxic = widget.dissolvedOxygen < 3.5;
    final isSeverelyHypoxic = widget.dissolvedOxygen < 2.5;
    final isAmmoniaToxic = widget.ammoniaTan > 1.5;
    final isCold = widget.temperature < 22.0;

    double speedMultiplier = 1.0;
    if (isCold) speedMultiplier = 0.5;
    if (isSeverelyHypoxic) speedMultiplier = 0.35;
    if (isAmmoniaToxic) speedMultiplier = 1.6;

    if (_isAutoRotating) {
      _rotY += 0.003;
    }

    // Update Bubbles
    for (var b in _bubbles) {
      b.y -= b.speed;
      b.wobblePhase += 0.1;
      b.x += math.sin(b.wobblePhase) * 0.002;
      if (b.y < 0.0) {
        b.y = 1.0;
        b.x = (_rng.nextDouble() - 0.5) * 0.8;
        b.z = (_rng.nextDouble() - 0.5) * 0.8;
      }
    }

    // Update Fishes
    for (var f in _fishes) {
      f.phase += 0.18 * speedMultiplier;

      // In hypoxia, fish seek the surface (y near 0.05) and pipe
      if (isHypoxic) {
        f.y += (0.05 - f.y) * 0.04;
      } else if (isCold) {
        // In cold weather, fish stay near bottom (y near 0.8)
        f.y += (0.75 - f.y) * 0.02;
      }

      f.x += f.vx * speedMultiplier;
      f.y += f.vy * speedMultiplier;
      f.z += f.vz * speedMultiplier;

      // Tank boundaries
      const boundX = 0.75;
      const boundZ = 0.75;
      if (f.x.abs() > boundX) {
        f.vx = -f.vx;
        f.x = f.x.clamp(-boundX, boundX);
      }
      if (f.z.abs() > boundZ) {
        f.vz = -f.vz;
        f.z = f.z.clamp(-boundZ, boundZ);
      }
      if (f.y < 0.02 || f.y > 0.95) {
        f.vy = -f.vy;
        f.y = f.y.clamp(0.02, 0.95);
      }

      // Smooth yaw rotation towards heading
      f.yaw = math.atan2(f.vz, f.vx);

      // Random gentle steering
      if (_rng.nextDouble() < 0.02) {
        final steer = (_rng.nextDouble() - 0.5) * 0.8;
        final spd = f.speed;
        f.yaw += steer;
        f.vx = math.cos(f.yaw) * spd;
        f.vz = math.sin(f.yaw) * spd;
      }
    }

    setState(() {});
  }

  void _setPreset(CameraPreset preset) {
    setState(() {
      _preset = preset;
      _isAutoRotating = false;
      switch (preset) {
        case CameraPreset.isometric:
          _rotX = 0.45;
          _rotY = -0.65;
          _zoom = 1.0;
          break;
        case CameraPreset.topDown:
          _rotX = math.pi / 2 - 0.05;
          _rotY = 0.0;
          _zoom = 1.1;
          break;
        case CameraPreset.surface:
          _rotX = 0.15;
          _rotY = -0.4;
          _zoom = 1.25;
          break;
        case CameraPreset.underwater:
          _rotX = -0.1;
          _rotY = -0.8;
          _zoom = 1.35;
          break;
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final isHypoxic = widget.dissolvedOxygen < 3.5;
    final isAmmoniaToxic = widget.ammoniaTan > 1.5;
    final isCold = widget.temperature < 22.0;

    String behaviorLabel = 'Optimal Cruising (28.4°C)';
    Color behaviorColor = Colors.greenAccent;
    if (isHypoxic) {
      behaviorLabel = '🚨 Hypoxic Surface Piping (DO < 3.5)';
      behaviorColor = Colors.redAccent;
    } else if (isAmmoniaToxic) {
      behaviorLabel = '⚡ Ammonia Agitation Stress';
      behaviorColor = Colors.orangeAccent;
    } else if (isCold) {
      behaviorLabel = '❄️ Cold-Water Sluggish Stupor (<22°C)';
      behaviorColor = Colors.lightBlueAccent;
    }

    return Container(
      height: 380,
      decoration: BoxDecoration(
        color: const Color(0xFF071118),
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: Colors.cyan.withOpacity(0.3), width: 1.5),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.5),
            blurRadius: 16,
            offset: const Offset(0, 6),
          ),
        ],
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(20),
        child: Stack(
          children: [
            // 3D Canvas
            GestureDetector(
              onPanUpdate: (details) {
                setState(() {
                  _isAutoRotating = false;
                  _rotY += details.delta.dx * 0.01;
                  _rotX = (_rotX - details.delta.dy * 0.01).clamp(-math.pi / 3, math.pi / 2.2);
                });
              },
              child: CustomPaint(
                painter: _CultureTank3DPainter(
                  rotX: _rotX,
                  rotY: _rotY,
                  zoom: _zoom,
                  fishes: _fishes,
                  bubbles: _bubbles,
                  dissolvedOxygen: widget.dissolvedOxygen,
                  temperature: widget.temperature,
                  systemType: widget.systemType,
                  animTime: _animController.value * 2 * math.pi,
                ),
                child: const SizedBox.expand(),
              ),
            ),

            // Top HUD Bar
            Positioned(
              top: 12,
              left: 14,
              right: 14,
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                    decoration: BoxDecoration(
                      color: Colors.black.withOpacity(0.65),
                      borderRadius: BorderRadius.circular(10),
                      border: Border.all(color: behaviorColor.withOpacity(0.5)),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(Icons.hub, color: behaviorColor, size: 14),
                        const SizedBox(width: 6),
                        Text(
                          behaviorLabel,
                          style: TextStyle(
                            color: behaviorColor,
                            fontSize: 11,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                      ],
                    ),
                  ),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                    decoration: BoxDecoration(
                      color: Colors.black.withOpacity(0.65),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Text(
                      '${_fishes.length} Animated Fish (${widget.avgWeightG.toInt()}g)',
                      style: const TextStyle(color: Colors.white70, fontSize: 10, fontWeight: FontWeight.bold),
                    ),
                  ),
                ],
              ),
            ),

            // Camera Presets & Auto-Rotate Controls
            Positioned(
              bottom: 12,
              left: 12,
              right: 12,
              child: Row(
                children: [
                  _PresetButton(
                    label: '3D Iso',
                    icon: Icons.view_in_ar,
                    isSelected: _preset == CameraPreset.isometric,
                    onTap: () => _setPreset(CameraPreset.isometric),
                  ),
                  const SizedBox(width: 6),
                  _PresetButton(
                    label: 'Top',
                    icon: Icons.vertical_align_top,
                    isSelected: _preset == CameraPreset.topDown,
                    onTap: () => _setPreset(CameraPreset.topDown),
                  ),
                  const SizedBox(width: 6),
                  _PresetButton(
                    label: 'Surface',
                    icon: Icons.waves,
                    isSelected: _preset == CameraPreset.surface,
                    onTap: () => _setPreset(CameraPreset.surface),
                  ),
                  const SizedBox(width: 6),
                  _PresetButton(
                    label: 'Deep',
                    icon: Icons.visibility,
                    isSelected: _preset == CameraPreset.underwater,
                    onTap: () => _setPreset(CameraPreset.underwater),
                  ),
                  const Spacer(),
                  IconButton.filledTonal(
                    style: IconButton.styleFrom(
                      padding: const EdgeInsets.all(6),
                      backgroundColor: _isAutoRotating ? Colors.cyan.withOpacity(0.3) : Colors.black45,
                      visualDensity: VisualDensity.compact,
                    ),
                    icon: Icon(
                      _isAutoRotating ? Icons.pause_circle_outline : Icons.play_circle_outline,
                      color: _isAutoRotating ? Colors.cyanAccent : Colors.white70,
                      size: 18,
                    ),
                    tooltip: 'Toggle 3D Orbit Rotation',
                    onPressed: () => setState(() => _isAutoRotating = !_isAutoRotating),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _PresetButton extends StatelessWidget {
  final String label;
  final IconData icon;
  final bool isSelected;
  final VoidCallback onTap;

  const _PresetButton({
    required this.label,
    required this.icon,
    required this.isSelected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(8),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 5),
        decoration: BoxDecoration(
          color: isSelected ? Colors.cyan.withOpacity(0.3) : Colors.black.withOpacity(0.6),
          borderRadius: BorderRadius.circular(8),
          border: Border.all(
            color: isSelected ? Colors.cyanAccent : Colors.white24,
            width: 1,
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 12, color: isSelected ? Colors.cyanAccent : Colors.white70),
            const SizedBox(width: 4),
            Text(
              label,
              style: TextStyle(
                color: isSelected ? Colors.cyanAccent : Colors.white70,
                fontSize: 10,
                fontWeight: isSelected ? FontWeight.bold : FontWeight.normal,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _CultureTank3DPainter extends CustomPainter {
  final double rotX;
  final double rotY;
  final double zoom;
  final List<Fish3D> fishes;
  final List<Bubble3D> bubbles;
  final double dissolvedOxygen;
  final double temperature;
  final String systemType;
  final double animTime;

  _CultureTank3DPainter({
    required this.rotX,
    required this.rotY,
    required this.zoom,
    required this.fishes,
    required this.bubbles,
    required this.dissolvedOxygen,
    required this.temperature,
    required this.systemType,
    required this.animTime,
  });

  // 3D to 2D projection
  Offset _project(double x, double y, double z, Size size) {
    // Center of tank is (0, 0, 0)
    // Scale up world coords
    final scale = 110.0 * zoom;

    // Rotate around Y axis
    final cosY = math.cos(rotY);
    final sinY = math.sin(rotY);
    final x1 = x * cosY + z * sinY;
    final z1 = -x * sinY + z * cosY;

    // Rotate around X axis
    final cosX = math.cos(rotX);
    final sinX = math.sin(rotX);
    final y2 = (y - 0.5) * cosX - z1 * sinX;
    final z2 = (y - 0.5) * sinX + z1 * cosX;

    // Perspective factor
    const cameraDist = 3.2;
    final pFactor = cameraDist / (cameraDist + z2);

    final screenX = size.width / 2 + x1 * scale * pFactor;
    final screenY = size.height / 2 + y2 * scale * pFactor + 10;
    return Offset(screenX, screenY);
  }

  double _getDepth(double x, double y, double z) {
    final cosY = math.cos(rotY);
    final sinY = math.sin(rotY);
    final z1 = -x * sinY + z * cosY;
    final sinX = math.sin(rotX);
    final cosX = math.cos(rotX);
    return (y - 0.5) * sinX + z1 * cosX;
  }

  @override
  void paint(Canvas canvas, Size size) {
    // 1. Draw 3D Tank Floor & Grid
    _drawTankStructure(canvas, size);

    // 2. Draw Aerator Diffuser Ring on bottom
    _drawAerationDiffuser(canvas, size);

    // 3. Collect all renderable entities for depth-sorting
    final List<_RenderItem> items = [];

    // Add Bubbles
    for (var b in bubbles) {
      items.add(_RenderItem(
        depth: _getDepth(b.x, b.y, b.z),
        draw: (c) => _drawBubble(c, b, size),
      ));
    }

    // Add Fish
    for (var f in fishes) {
      items.add(_RenderItem(
        depth: _getDepth(f.x, f.y, f.z),
        draw: (c) => _drawFish(c, f, size),
      ));
    }

    // Sort back-to-front
    items.sort((a, b) => b.depth.compareTo(a.depth));

    for (var item in items) {
      item.draw(canvas);
    }

    // 4. Draw Water Surface Shimmer & Front Glass Walls
    _drawWaterSurfaceAndGlass(canvas, size);
  }

  void _drawTankStructure(Canvas canvas, Size size) {
    const half = 0.85;

    // Tank bottom vertices (y = 1.0)
    final b0 = _project(-half, 1.0, -half, size);
    final b1 = _project(half, 1.0, -half, size);
    final b2 = _project(half, 1.0, half, size);
    final b3 = _project(-half, 1.0, half, size);

    final floorPath = Path()
      ..moveTo(b0.dx, b0.dy)
      ..lineTo(b1.dx, b1.dy)
      ..lineTo(b2.dx, b2.dy)
      ..lineTo(b3.dx, b3.dy)
      ..close();

    // Fill Floor
    final floorPaint = Paint()
      ..shader = LinearGradient(
        colors: [
          const Color(0xFF0B1924),
          const Color(0xFF0F2B3C),
        ],
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
      ).createShader(Rect.fromLTWH(0, 0, size.width, size.height));
    canvas.drawPath(floorPath, floorPaint);

    // Floor Grid Lines
    final gridPaint = Paint()
      ..color = Colors.cyanAccent.withOpacity(0.12)
      ..strokeWidth = 1.0;
    for (double i = -half; i <= half; i += 0.34) {
      canvas.drawLine(_project(i, 1.0, -half, size), _project(i, 1.0, half, size), gridPaint);
      canvas.drawLine(_project(-half, 1.0, i, size), _project(half, 1.0, i, size), gridPaint);
    }

    // Tank Pillar Edges (y = 0.0 to y = 1.0)
    final pTop0 = _project(-half, 0.0, -half, size);
    final pTop1 = _project(half, 0.0, -half, size);
    final pTop2 = _project(half, 0.0, half, size);
    final pTop3 = _project(-half, 0.0, half, size);

    final wallEdgePaint = Paint()
      ..color = Colors.cyan.withOpacity(0.35)
      ..strokeWidth = 1.5;

    canvas.drawLine(b0, pTop0, wallEdgePaint);
    canvas.drawLine(b1, pTop1, wallEdgePaint);
    canvas.drawLine(b2, pTop2, wallEdgePaint);
    canvas.drawLine(b3, pTop3, wallEdgePaint);

    // Back walls shading
    final backWallPath = Path()
      ..moveTo(b0.dx, b0.dy)
      ..lineTo(b1.dx, b1.dy)
      ..lineTo(pTop1.dx, pTop1.dy)
      ..lineTo(pTop0.dx, pTop0.dy)
      ..close();
    canvas.drawPath(backWallPath, Paint()..color = const Color(0xFF091620).withOpacity(0.6));
  }

  void _drawAerationDiffuser(Canvas canvas, Size size) {
    // Ring on bottom (y = 0.98)
    final diffPaint = Paint()
      ..color = Colors.cyanAccent.withOpacity(0.4)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.0;

    final ringPath = Path();
    const radius = 0.45;
    for (int i = 0; i <= 36; i++) {
      final angle = (i / 36) * 2 * math.pi;
      final pt = _project(math.cos(angle) * radius, 0.98, math.sin(angle) * radius, size);
      if (i == 0) {
        ringPath.moveTo(pt.dx, pt.dy);
      } else {
        ringPath.lineTo(pt.dx, pt.dy);
      }
    }
    canvas.drawPath(ringPath, diffPaint);
  }

  void _drawFish(Canvas canvas, Fish3D fish, Size size) {
    final headPt = _project(fish.x, fish.y, fish.z, size);

    // Calculate tail point with sinusoidal oscillation
    final tailWiggle = math.sin(fish.phase) * 0.08;
    final tailAngle = fish.yaw + math.pi + tailWiggle;
    final tailX = fish.x + math.cos(tailAngle) * fish.length;
    final tailZ = fish.z + math.sin(tailAngle) * fish.length;
    final tailPt = _project(tailX, fish.y, tailZ, size);

    // Midbody point
    final midX = (fish.x + tailX) / 2 + math.cos(fish.yaw + math.pi / 2) * (tailWiggle * 0.5);
    final midZ = (fish.z + tailZ) / 2 + math.sin(fish.yaw + math.pi / 2) * (tailWiggle * 0.5);
    final midPt = _project(midX, fish.y, midZ, size);

    // Draw Fish Body
    final bodyPaint = Paint()
      ..color = fish.color
      ..strokeWidth = 4.5 * zoom
      ..strokeCap = StrokeCap.round;
    canvas.drawLine(headPt, midPt, bodyPaint);

    final tailPaint = Paint()
      ..color = fish.color.withOpacity(0.85)
      ..strokeWidth = 3.0 * zoom
      ..strokeCap = StrokeCap.round;
    canvas.drawLine(midPt, tailPt, tailPaint);

    // Tail Fin (Fan)
    final fin1Pt = _project(tailX + math.cos(tailAngle + 0.5) * 0.05, fish.y - 0.02, tailZ + math.sin(tailAngle + 0.5) * 0.05, size);
    final fin2Pt = _project(tailX + math.cos(tailAngle - 0.5) * 0.05, fish.y + 0.02, tailZ + math.sin(tailAngle - 0.5) * 0.05, size);
    final finPath = Path()
      ..moveTo(tailPt.dx, tailPt.dy)
      ..lineTo(fin1Pt.dx, fin1Pt.dy)
      ..lineTo(fin2Pt.dx, fin2Pt.dy)
      ..close();
    canvas.drawPath(finPath, Paint()..color = Colors.blueGrey.withOpacity(0.7));

    // Barbels / Catfish whiskers if head is near
    final w1 = _project(fish.x + math.cos(fish.yaw + 0.4) * 0.04, fish.y + 0.01, fish.z + math.sin(fish.yaw + 0.4) * 0.04, size);
    final w2 = _project(fish.x + math.cos(fish.yaw - 0.4) * 0.04, fish.y + 0.01, fish.z + math.sin(fish.yaw - 0.4) * 0.04, size);
    final whiskerPaint = Paint()
      ..color = Colors.white54
      ..strokeWidth = 1.0;
    canvas.drawLine(headPt, w1, whiskerPaint);
    canvas.drawLine(headPt, w2, whiskerPaint);
  }

  void _drawBubble(Canvas canvas, Bubble3D b, Size size) {
    final pt = _project(b.x, b.y, b.z, size);
    final bubblePaint = Paint()
      ..color = Colors.cyanAccent.withOpacity(0.4)
      ..style = PaintingStyle.fill;
    canvas.drawCircle(pt, b.size * zoom, bubblePaint);

    final highlightPaint = Paint()
      ..color = Colors.white.withOpacity(0.7)
      ..style = PaintingStyle.fill;
    canvas.drawCircle(Offset(pt.dx - b.size * 0.3, pt.dy - b.size * 0.3), b.size * 0.35, highlightPaint);
  }

  void _drawWaterSurfaceAndGlass(Canvas canvas, Size size) {
    const half = 0.85;

    // Water Surface Plane (y = 0.05)
    final s0 = _project(-half, 0.05, -half, size);
    final s1 = _project(half, 0.05, -half, size);
    final s2 = _project(half, 0.05, half, size);
    final s3 = _project(-half, 0.05, half, size);

    final surfacePath = Path()
      ..moveTo(s0.dx, s0.dy)
      ..lineTo(s1.dx, s1.dy)
      ..lineTo(s2.dx, s2.dy)
      ..lineTo(s3.dx, s3.dy)
      ..close();

    final surfacePaint = Paint()
      ..color = const Color(0xFF00E5FF).withOpacity(0.14)
      ..style = PaintingStyle.fill;
    canvas.drawPath(surfacePath, surfacePaint);

    final surfaceRimPaint = Paint()
      ..color = Colors.cyanAccent.withOpacity(0.5)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.2;
    canvas.drawPath(surfacePath, surfaceRimPaint);

    // Front Glass Wall Outline
    final bTop0 = _project(-half, 0.0, half, size);
    final bTop1 = _project(half, 0.0, half, size);
    final bBot0 = _project(-half, 1.0, half, size);
    final bBot1 = _project(half, 1.0, half, size);

    final frontGlassPath = Path()
      ..moveTo(bTop0.dx, bTop0.dy)
      ..lineTo(bTop1.dx, bTop1.dy)
      ..lineTo(bBot1.dx, bBot1.dy)
      ..lineTo(bBot0.dx, bBot0.dy)
      ..close();

    final glassPaint = Paint()
      ..color = Colors.cyan.withOpacity(0.04)
      ..style = PaintingStyle.fill;
    canvas.drawPath(frontGlassPath, glassPaint);

    canvas.drawLine(bTop0, bTop1, Paint()..color = Colors.cyanAccent.withOpacity(0.6)..strokeWidth = 1.5);
  }

  @override
  bool shouldRepaint(covariant _CultureTank3DPainter oldDelegate) => true;
}

class _RenderItem {
  final double depth;
  final void Function(Canvas) draw;

  _RenderItem({required this.depth, required this.draw});
}
