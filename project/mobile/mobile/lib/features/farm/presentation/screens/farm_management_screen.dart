import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:http/http.dart' as http;
import '../../../core/constants/app_colors.dart';
import '../../../core/providers/auth_provider.dart';

class FarmManagementScreen extends ConsumerStatefulWidget {
  const FarmManagementScreen({super.key});

  @override
  ConsumerState<FarmManagementScreen> createState() => _FarmManagementScreenState();
}

class _FarmManagementScreenState extends ConsumerState<FarmManagementScreen> {
  bool _isLoading = true;
  List<dynamic> _farms = [];
  Map<int, List<dynamic>> _unitsByFarm = {};

  @override
  void initState() {
    super.initState();
    _fetchFarms();
  }

  Future<void> _fetchFarms() async {
    setState(() => _isLoading = true);
    try {
      final auth = ref.read(authStateProvider);
      final token = auth.token;

      final response = await http.get(
        Uri.parse('http://localhost:8080/api/v1/farms'),
        headers: {
          'Content-Type': 'application/json',
          if (token != null) 'Authorization': 'Bearer $token',
        },
      );

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body) as List;
        setState(() {
          _farms = data;
        });

        for (var f in data) {
          final farmId = f['id'] as int;
          final units = (f['production_units'] as List? ?? []);
          _unitsByFarm[farmId] = units;
        }
      }
    } catch (e) {
      // Fallback mock demonstration
      _farms = [
        {
          'id': 1,
          'name': 'Delta Fish Estate - Main Facility',
          'location': 'Warri, Delta State',
        }
      ];
      _unitsByFarm[1] = [
        {
          'id': 101,
          'name': 'Earthen Pond 1 (Catfish)',
          'unit_type': 'earthen_pond',
          'volume_liters': 60000,
          'max_biomass_kg': 1200,
          'status': 'active',
        },
        {
          'id': 102,
          'name': 'Concrete Tank Alpha (Nursery)',
          'unit_type': 'concrete_tank',
          'volume_liters': 25000,
          'max_biomass_kg': 600,
          'status': 'active',
        },
      ];
    } finally {
      setState(() => _isLoading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Farm Management', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
            Text('Precision Ponds, Tanks & Cohorts', style: TextStyle(fontSize: 11, color: Colors.grey)),
          ],
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: _fetchFarms,
          ),
        ],
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : _farms.isEmpty
              ? _buildEmptyState()
              : ListView.builder(
                  padding: const EdgeInsets.all(16),
                  itemCount: _farms.length,
                  itemBuilder: (context, index) {
                    final farm = _farms[index];
                    final farmId = farm['id'] as int;
                    final units = _unitsByFarm[farmId] ?? [];
                    return _buildFarmCard(farm, units);
                  },
                ),
      floatingActionButton: FloatingActionButton.extended(
        backgroundColor: AppColors.primary,
        icon: const Icon(Icons.add, color: Colors.white),
        label: const Text('New Pond / Tank', style: TextStyle(color: Colors.white)),
        onPressed: () => _showAddUnitDialog(),
      ),
    );
  }

  Widget _buildFarmCard(dynamic farm, List<dynamic> units) {
    return Card(
      margin: const EdgeInsets.only(bottom: 20),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      elevation: 2,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Farm Title
            Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: AppColors.primary.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: const Icon(Icons.water, color: AppColors.primary, size: 24),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(farm['name'] ?? 'Aquaculture Facility', style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
                      if (farm['location'] != null)
                        Text(farm['location'], style: TextStyle(fontSize: 12, color: Colors.grey[600])),
                    ],
                  ),
                ),
                Chip(
                  label: Text('${units.length} Units', style: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold)),
                  backgroundColor: Colors.blue.withOpacity(0.1),
                ),
              ],
            ),
            const SizedBox(height: 16),
            const Divider(),
            const SizedBox(height: 8),
            // Units Grid / List
            const Text('Production Units', style: TextStyle(fontSize: 13, fontWeight: FontWeight.bold, color: Colors.grey)),
            const SizedBox(height: 10),
            ...units.map((u) => _buildUnitTile(u)),
          ],
        ),
      ),
    );
  }

  Widget _buildUnitTile(dynamic unit) {
    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.grey.withOpacity(0.06),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: Colors.grey.withOpacity(0.15)),
      ),
      child: Row(
        children: [
          Icon(
            _getUnitIcon(unit['unit_type']),
            color: AppColors.primary,
            size: 28,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(unit['name'] ?? 'Production Unit', style: const TextStyle(fontSize: 14, fontWeight: FontWeight.bold)),
                const SizedBox(height: 2),
                Text(
                  '${(unit['volume_liters'] ?? 0) / 1000} m³ • Max Biomass: ${unit['max_biomass_kg'] ?? 'N/A'} kg',
                  style: TextStyle(fontSize: 11, color: Colors.grey[700]),
                ),
              ],
            ),
          ),
          OutlinedButton(
            style: OutlinedButton.styleFrom(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              visualDensity: VisualDensity.compact,
            ),
            onPressed: () => _showUnitQuickActions(unit),
            child: const Text('Manage', style: TextStyle(fontSize: 12)),
          ),
        ],
      ),
    );
  }

  IconData _getUnitIcon(String? type) {
    switch (type) {
      case 'earthen_pond':
        return Icons.waves;
      case 'concrete_tank':
        return Icons.crop_square;
      case 'plastic_tank':
      case 'tarpaulin_tank':
        return Icons.circle_outlined;
      case 'ras_tank':
        return Icons.autorenew;
      case 'cage':
        return Icons.grid_4x4;
      default:
        return Icons.water_drop;
    }
  }

  Widget _buildEmptyState() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.waves, size: 64, color: Colors.grey[400]),
          const SizedBox(height: 16),
          const Text('No Production Units Found', style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
          const SizedBox(height: 8),
          Text('Add your earthen ponds, concrete tanks, or cages to begin.', style: TextStyle(fontSize: 13, color: Colors.grey[600])),
        ],
      ),
    );
  }

  void _showAddUnitDialog() {
    final nameCtrl = TextEditingController();
    final volumeCtrl = TextEditingController();

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Add Production Unit'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(controller: nameCtrl, decoration: const InputDecoration(labelText: 'Unit Name (e.g. Pond 2)')),
            const SizedBox(height: 10),
            TextField(controller: volumeCtrl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: 'Volume (Liters)')),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          ElevatedButton(
            onPressed: () {
              Navigator.pop(ctx);
              _fetchFarms();
            },
            child: const Text('Save'),
          ),
        ],
      ),
    );
  }

  void _showUnitQuickActions(dynamic unit) {
    showModalBottomSheet(
      context: context,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
      builder: (ctx) => Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(unit['name'] ?? 'Unit Details', style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
            const SizedBox(height: 14),
            ListTile(
              leading: const Icon(Icons.scale, color: Colors.blue),
              title: const Text('Log Sampling Biometrics'),
              subtitle: const Text('Record sample average weight (g) and condition factor'),
              onTap: () => Navigator.pop(ctx),
            ),
            ListTile(
              leading: const Icon(Icons.remove_circle_outline, color: Colors.red),
              title: const Text('Record Mortality Event'),
              subtitle: const Text('Deduct count from active cohort biomass'),
              onTap: () => Navigator.pop(ctx),
            ),
            ListTile(
              leading: const Icon(Icons.psychology, color: AppColors.primary),
              title: const Text('Consult AquaDoc for this Pond'),
              subtitle: const Text('Analyze live water parameters & disease risks'),
              onTap: () => Navigator.pop(ctx),
            ),
          ],
        ),
      ),
    );
  }
}
