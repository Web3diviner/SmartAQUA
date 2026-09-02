import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/models/feeding.dart';
import '../../../../core/providers/calculator_provider.dart';
import '../../../../core/providers/monitoring_provider.dart';

class FeedCalculatorScreen extends ConsumerStatefulWidget {
  const FeedCalculatorScreen({super.key});

  @override
  ConsumerState<FeedCalculatorScreen> createState() =>
      _FeedCalculatorScreenState();
}

class _FeedCalculatorScreenState extends ConsumerState<FeedCalculatorScreen> {
  String? _selectedSpeciesId;
  double _fishCount = 1000;
  double _avgWeight = 200;
  double _waterTemp = 28;
  final Map<String, TextEditingController> _inputControllers = {};
  final Map<String, FocusNode> _inputFocusNodes = {};

  @override
  void initState() {
    super.initState();
    Future.microtask(_loadData);
  }

  @override
  void dispose() {
    for (final controller in _inputControllers.values) {
      controller.dispose();
    }
    for (final focusNode in _inputFocusNodes.values) {
      focusNode.dispose();
    }
    super.dispose();
  }

  Future<void> _loadData() async {
    await ref.read(speciesListProvider.notifier).loadSpecies();
    if (!mounted) return;

    final species = ref.read(speciesProvider);
    if (species.isNotEmpty && _selectedSpeciesId == null) {
      setState(() => _selectedSpeciesId = species.first.id);
    }

    // Try to get current sensor data for temperature
    final sensorData = ref.read(sensorDataProvider).currentData;
    if (sensorData != null) {
      setState(() {
        _waterTemp = sensorData.waterTemperature;
      });
    }
  }

  Future<void> _calculate() async {
    if (_selectedSpeciesId == null) return;

    FocusScope.of(context).unfocus();

    final request = FeedCalculationRequest(
      speciesId: _selectedSpeciesId!,
      fishCount: _fishCount.round(),
      averageWeight: _avgWeight,
      waterTemperature: _waterTemp,
    );

    await ref.read(calculatorProvider.notifier).calculate(request);
  }

  void _updateInput(VoidCallback updater) {
    setState(updater);
    ref.read(calculatorProvider.notifier).clearResult();
  }

  String _formatFieldValue(double value, {required bool isInteger}) {
    return isInteger ? value.round().toString() : value.toStringAsFixed(1);
  }

  TextEditingController _getInputController(String fieldId, String text) {
    return _inputControllers.putIfAbsent(
      fieldId,
      () => TextEditingController(text: text),
    );
  }

  FocusNode _getInputFocusNode(String fieldId) {
    return _inputFocusNodes.putIfAbsent(fieldId, FocusNode.new);
  }

  void _syncInputText({required String fieldId, required String text}) {
    final controller = _getInputController(fieldId, text);
    final focusNode = _getInputFocusNode(fieldId);
    if (focusNode.hasFocus || controller.text == text) return;
    controller.value = controller.value.copyWith(
      text: text,
      selection: TextSelection.collapsed(offset: text.length),
      composing: TextRange.empty,
    );
  }

  @override
  Widget build(BuildContext context) {
    final speciesState = ref.watch(speciesListProvider);
    final calcState = ref.watch(calculatorProvider);
    final result = calcState.result;

    return Scaffold(
      appBar: AppBar(title: const Text('Feed Calculator')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            // Species selector
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Fish Species',
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 8),
                    speciesState.isLoading
                        ? const Center(child: CircularProgressIndicator())
                        : speciesState.error != null
                        ? Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              speciesState.error!,
                              style: TextStyle(color: Colors.red.shade700),
                            ),
                            const SizedBox(height: 12),
                            FilledButton.tonalIcon(
                              onPressed: () {
                                _loadData();
                              },
                              icon: const Icon(Icons.refresh),
                              label: const Text('Retry species load'),
                            ),
                          ],
                        )
                        : speciesState.species.isEmpty
                        ? Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            const Text(
                              'No fish species are available from the backend yet.',
                            ),
                            const SizedBox(height: 12),
                            FilledButton.tonalIcon(
                              onPressed: () {
                                _loadData();
                              },
                              icon: const Icon(Icons.refresh),
                              label: const Text('Reload species'),
                            ),
                          ],
                        )
                        : DropdownButtonFormField<String>(
                          key: ValueKey(_selectedSpeciesId),
                          initialValue: _selectedSpeciesId,
                          items:
                              speciesState.species
                                  .map(
                                    (s) => DropdownMenuItem(
                                      value: s.id,
                                      child: Text(s.name),
                                    ),
                                  )
                                  .toList(),
                          onChanged:
                              (v) => _updateInput(() => _selectedSpeciesId = v),
                          decoration: const InputDecoration(
                            border: OutlineInputBorder(),
                          ),
                        ),
                    if (_selectedSpeciesId != null) ...[
                      const SizedBox(height: 8),
                      _buildSpeciesInfo(speciesState.species),
                    ],
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),

            // Parameters
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _buildSlider(
                      fieldId: 'fishCount',
                      label: 'Fish Count',
                      value: _fishCount,
                      min: 1,
                      max: 10000,
                      divisions: 200,
                      suffix: ' fish',
                      isInteger: true,
                      onChanged: (v) => _updateInput(() => _fishCount = v),
                    ),
                    const SizedBox(height: 16),
                    _buildSlider(
                      fieldId: 'averageWeight',
                      label: 'Average Weight',
                      value: _avgWeight,
                      min: 1,
                      max: 1000,
                      divisions: 200,
                      suffix: 'g',
                      onChanged: (v) => _updateInput(() => _avgWeight = v),
                    ),
                    const SizedBox(height: 16),
                    _buildSlider(
                      fieldId: 'waterTemperature',
                      label: 'Water Temperature',
                      value: _waterTemp,
                      min: 0,
                      max: 35,
                      divisions: 70,
                      suffix: '°C',
                      onChanged: (v) => _updateInput(() => _waterTemp = v),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),

            // Calculate button
            FilledButton(
              onPressed: calcState.isCalculating ? null : _calculate,
              child:
                  calcState.isCalculating
                      ? const SizedBox(
                        height: 20,
                        width: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                      : const Text('Calculate'),
            ),

            // Error message
            if (calcState.error != null)
              Padding(
                padding: const EdgeInsets.only(top: 16),
                child: Card(
                  color: Colors.red.shade50,
                  child: Padding(
                    padding: const EdgeInsets.all(12),
                    child: Text(
                      calcState.error!,
                      style: TextStyle(color: Colors.red.shade700),
                    ),
                  ),
                ),
              ),

            // Result
            if (result != null) ...[
              const SizedBox(height: 24),
              Card(
                color: Theme.of(context).colorScheme.primaryContainer,
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Column(
                    children: [
                      Text(
                        'Recommended Daily Feed',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 8),
                      Text(
                        '${result.recommendedAmount.round()}g',
                        style: Theme.of(
                          context,
                        ).textTheme.displayMedium?.copyWith(
                          fontWeight: FontWeight.bold,
                          color: Theme.of(context).colorScheme.primary,
                        ),
                      ),
                      const SizedBox(height: 16),
                      _buildResultRow(
                        'Biomass',
                        '${result.biomass.toStringAsFixed(1)} kg',
                      ),
                      _buildResultRow(
                        'Feeding Rate (BW/day)',
                        '${(result.feedingRate * 100).toStringAsFixed(2)}%',
                      ),
                      _buildResultRow(
                        'Q10 Factor',
                        result.q10Factor.toStringAsFixed(2),
                      ),
                      if (result.obmSafetyFactor != null)
                        _buildResultRow(
                          'OBM Safety',
                          '${(result.obmSafetyFactor! * 100).toStringAsFixed(0)}%',
                        ),
                      _buildResultRow(
                        'Suggested Feedings',
                        '${result.suggestedFeedings}/day',
                      ),
                      const SizedBox(height: 12),
                      Text(
                        result.recommendation,
                        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          color: Colors.grey[700],
                        ),
                        textAlign: TextAlign.center,
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildSpeciesInfo(List<FishSpecies> species) {
    final selected =
        species.where((s) => s.id == _selectedSpeciesId).firstOrNull;
    if (selected == null) return const SizedBox.shrink();

    return Container(
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        color: Colors.grey.shade100,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceAround,
        children: [
          _buildInfoChip(
            'Base BW%',
            '${selected.feedingRatePercentage.toStringAsFixed(1)}%',
          ),
          _buildInfoChip('Q10', selected.q10Coefficient.toString()),
          _buildInfoChip(
            'Optimal',
            '${selected.optimalTempMin.toInt()}-${selected.optimalTempMax.toInt()}°C',
          ),
          _buildInfoChip(
            'Ref Temp',
            '${selected.referenceTemperature.toInt()}°C',
          ),
        ],
      ),
    );
  }

  Widget _buildInfoChip(String label, String value) {
    return Column(
      children: [
        Text(value, style: const TextStyle(fontWeight: FontWeight.bold)),
        Text(label, style: TextStyle(fontSize: 12, color: Colors.grey[600])),
      ],
    );
  }

  Widget _buildSlider({
    required String fieldId,
    required String label,
    required double value,
    required double min,
    required double max,
    required int divisions,
    required String suffix,
    bool isInteger = false,
    required ValueChanged<double> onChanged,
  }) {
    final displayValue = _formatFieldValue(value, isInteger: isInteger);
    _syncInputText(fieldId: fieldId, text: displayValue);
    final controller = _getInputController(fieldId, displayValue);
    final focusNode = _getInputFocusNode(fieldId);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label, style: Theme.of(context).textTheme.titleMedium),
            SizedBox(
              width: 110,
              child: TextFormField(
                controller: controller,
                focusNode: focusNode,
                textAlign: TextAlign.right,
                keyboardType: TextInputType.numberWithOptions(
                  decimal: !isInteger,
                ),
                decoration: InputDecoration(
                  isDense: true,
                  suffixText: suffix.trim(),
                  border: const OutlineInputBorder(),
                ),
                onChanged: (text) {
                  final parsed = double.tryParse(text.trim());
                  if (parsed == null) return;
                  final clamped = parsed.clamp(min, max).toDouble();
                  final nextValue =
                      isInteger ? clamped.roundToDouble() : clamped;
                  if (nextValue == value) return;
                  onChanged(nextValue);
                },
                onFieldSubmitted: (text) {
                  final parsed = double.tryParse(text.trim());
                  if (parsed == null) return;
                  final clamped = parsed.clamp(min, max).toDouble();
                  onChanged(isInteger ? clamped.roundToDouble() : clamped);
                },
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        Slider(
          value: value,
          min: min,
          max: max,
          divisions: divisions,
          onChanged: onChanged,
        ),
      ],
    );
  }

  Widget _buildResultRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(label, style: TextStyle(color: Colors.grey[600])),
          Text(value, style: const TextStyle(fontWeight: FontWeight.w500)),
        ],
      ),
    );
  }
}
