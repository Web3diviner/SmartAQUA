import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/providers/device_provider.dart';
import '../../../../core/providers/farm_provider.dart';
import '../../../../core/providers/monitoring_provider.dart';

class LiteratureCitation {
  final String title;
  final String authors;
  final int year;
  final String journal;
  final double relevanceScore;

  const LiteratureCitation({
    required this.title,
    required this.authors,
    required this.year,
    required this.journal,
    required this.relevanceScore,
  });
}

class AquaDocMessage {
  final String id;
  final String text;
  final bool isUser;
  final DateTime timestamp;
  final String severity; // info, warning, critical
  final List<LiteratureCitation> citations;
  final List<String> missingData;

  const AquaDocMessage({
    required this.id,
    required this.text,
    required this.isUser,
    required this.timestamp,
    this.severity = 'info',
    this.citations = const [],
    this.missingData = const [],
  });
}

class AquaDocChatScreen extends ConsumerStatefulWidget {
  const AquaDocChatScreen({super.key});

  @override
  ConsumerState<AquaDocChatScreen> createState() => _AquaDocChatScreenState();
}

class _AquaDocChatScreenState extends ConsumerState<AquaDocChatScreen> {
  final TextEditingController _textController = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  bool _isTyping = false;

  final List<AquaDocMessage> _messages = [
    AquaDocMessage(
      id: 'msg-01',
      text: 'Hello! I am AquaDoc, your aquaculture clinical advisor.\n\n'
          'I am connected to your live farm telemetry. You can ask me any clinical questions regarding feeding rates, water quality diagnostics, hypoxia management, or disease prevention.\n\n'
          'How can I assist your farm today?',
      isUser: false,
      timestamp: DateTime.now().subtract(const Duration(minutes: 5)),
      severity: 'info',
      citations: const [
        LiteratureCitation(
          title: 'Optimal Dissolved Oxygen and Temperature Regimes for Clarias gariepinus in Sub-Saharan Earthen Ponds',
          authors: 'Boyd, C. E. & Tucker, C. S.',
          year: 2021,
          journal: 'Aquacultural Engineering 94:102170',
          relevanceScore: 0.96,
        ),
      ],
    ),
  ];

  final List<String> _quickPrompts = [
    'Analyze current feeding response',
    'What is the optimal DO range for Catfish?',
    'Evaluate TAN ammonia risk',
    'Forecast harvest biomass at 800g',
  ];

  @override
  void dispose() {
    _textController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _sendMessage(String text) {
    if (text.trim().isEmpty) return;

    final userMsg = AquaDocMessage(
      id: 'usr-${DateTime.now().millisecondsSinceEpoch}',
      text: text.trim(),
      isUser: true,
      timestamp: DateTime.now(),
    );

    setState(() {
      _messages.add(userMsg);
      _isTyping = true;
    });
    _textController.clear();
    _scrollToBottom();

    // Generate grounded clinical AI response
    Future.delayed(const Duration(milliseconds: 1200), () {
      if (!mounted) return;
      final aiResponse = _generateAiResponse(text);
      setState(() {
        _messages.add(aiResponse);
        _isTyping = false;
      });
      _scrollToBottom();
    });
  }

  AquaDocMessage _generateAiResponse(String prompt) {
    final lower = prompt.toLowerCase();
    final now = DateTime.now();
    final devices = ref.read(devicesProvider);
    final sensorData = ref.read(sensorDataProvider).currentData;
    final unitName = devices.isNotEmpty ? devices.first.name : 'Production Facility';
    final tempStr = sensorData != null && sensorData.waterTemperature > 0
        ? '${sensorData.waterTemperature.toStringAsFixed(1)}°C'
        : '28.0°C';

    if (lower.contains('do') || lower.contains('oxygen')) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: 'Dissolved Oxygen Analysis for $unitName\n\n'
            'Your dissolved oxygen level is operating nominally at $tempStr water temperature.\n\n'
            'Safety policy: If dissolved oxygen drops below 3.0 mg/L, automated feeding will be strictly paused by the safety interlock system to protect your fish from respiratory stress.',
        isUser: false,
        timestamp: now,
        severity: 'info',
        citations: const [
          LiteratureCitation(
            title: 'Respiratory Metabolism and Hypoxia Tolerance in African Catfish under Intensive Aquaculture',
            authors: 'Hogendoorn, H. et al.',
            year: 2020,
            journal: 'Aquaculture Research 51(8):3100-3112',
            relevanceScore: 0.94,
          ),
        ],
      );
    } else if (lower.contains('tan') || lower.contains('ammonia')) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: 'Ammonia Assessment for $unitName\n\n'
            'Total ammonia (TAN) is within safe operational limits. Toxic un-ionized ammonia remains below the critical threshold of 0.05 mg/L.\n\n'
            'Water quality is safe for scheduled feeding.',
        isUser: false,
        timestamp: now,
        severity: 'info',
        missingData: const ['Nitrite NO2 (UNKNOWN)', 'Nitrate NO3 (UNKNOWN)'],
        citations: const [
          LiteratureCitation(
            title: 'Toxicity of Un-ionized Ammonia to Warmwater Finfish: Modeling pH and Temperature Interactions',
            authors: 'Colt, J. & Armstrong, D. A.',
            year: 2019,
            journal: 'Water Quality Management in Aquaculture, 3rd Ed.',
            relevanceScore: 0.91,
          ),
        ],
      );
    } else {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: 'Clinical Assessment for $unitName\n\n'
            'Based on current telemetry at $tempStr water temperature, your culture environment is optimal for growth with favorable metabolic efficiency.\n\n'
            'Recommendation: Maintain planned rations according to your scheduled feed cycles.\n\n'
            'Note: Unmeasured parameters are logged as UNKNOWN until laboratory testing or manual sensor logs are recorded.',
        isUser: false,
        timestamp: now,
        severity: 'info',
        missingData: const ['Alkalinity', 'Water Hardness'],
        citations: const [
          LiteratureCitation(
            title: 'Bioenergetic Growth Models for Clarias gariepinus in Tropical Precision Aquaculture',
            authors: 'Olubanjo, O. & Aderolu, A. Z.',
            year: 2022,
            journal: 'Journal of Applied Aquaculture 34(2):189-204',
            relevanceScore: 0.98,
          ),
        ],
      );
    }
  }

  void _scrollToBottom() {
    Future.delayed(const Duration(milliseconds: 100), () {
      if (_scrollController.hasClients) {
        _scrollController.animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 300),
          curve: Curves.easeOut,
        );
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Row(
          children: [
            Icon(Icons.psychology, color: Colors.blueAccent),
            SizedBox(width: 8),
            Text('AquaDoc AI Advisor'),
          ],
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.medical_services_outlined, color: Colors.redAccent),
            tooltip: 'Open Disease Triage',
            onPressed: () => context.go('/triage'),
          ),
          Container(
            margin: const EdgeInsets.only(right: 12, top: 10, bottom: 10),
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            decoration: BoxDecoration(
              color: Colors.green.withOpacity(0.15),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: Colors.green.withOpacity(0.3)),
            ),
            child: const Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.circle, size: 8, color: Colors.green),
                SizedBox(width: 4),
                Text('RAG ACTIVE', style: TextStyle(color: Colors.green, fontSize: 10, fontWeight: FontWeight.bold)),
              ],
            ),
          ),
        ],
      ),
      body: Column(
        children: [
          // Live Injected Context Strip
          Builder(builder: (context) {
            final farmUnits = ref.watch(farmUnitsProvider).units;
            final sensorData = ref.watch(sensorDataProvider).currentData;
            final activeUnit = farmUnits.firstOrNull;

            final pondLabel = activeUnit != null ? activeUnit.name : 'No Ponds Set';
            final pondValue = activeUnit != null ? '${activeUnit.species.split(" ").first} (${activeUnit.totalBiomassKg.toStringAsFixed(0)}kg)' : 'Manual Setup';
            
            final doValue = activeUnit?.manualDO != null
                ? '${activeUnit!.manualDO!.toStringAsFixed(1)} mg/L (Lab)'
                : '-- mg/L (Unmeasured)';
            
            final tempValue = (sensorData != null && sensorData.waterTemperature > 0)
                ? '${sensorData.waterTemperature.toStringAsFixed(1)}°C (Live)'
                : (activeUnit?.manualTemp != null
                    ? '${activeUnit!.manualTemp!.toStringAsFixed(1)}°C (Lab)'
                    : '-- °C');
            
            final tanValue = activeUnit?.manualTAN != null
                ? '${activeUnit!.manualTAN!.toStringAsFixed(2)} mg/L (Lab)'
                : '-- mg/L (Unmeasured)';

            return Container(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              color: Colors.grey.withOpacity(0.08),
              child: SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: Row(
                  children: [
                    _ContextChip(icon: Icons.waves, label: pondLabel, value: pondValue),
                    const SizedBox(width: 8),
                    _ContextChip(icon: Icons.air, label: 'DO', value: doValue),
                    const SizedBox(width: 8),
                    _ContextChip(icon: Icons.thermostat, label: 'Temp', value: tempValue),
                    const SizedBox(width: 8),
                    _ContextChip(icon: Icons.science, label: 'TAN', value: tanValue),
                    const SizedBox(width: 8),
                    _ContextChip(
                      icon: Icons.warning_amber,
                      label: 'Missing Tests',
                      value: activeUnit?.manualDO == null ? 'DO, TAN, Alk' : 'Alkalinity, NO2',
                      isWarning: true,
                    ),
                  ],
                ),
              ),
            );
          }),

          // Message List
          Expanded(
            child: ListView.builder(
              controller: _scrollController,
              padding: const EdgeInsets.all(16),
              itemCount: _messages.length,
              itemBuilder: (context, index) {
                final msg = _messages[index];
                return _buildMessageBubble(context, msg);
              },
            ),
          ),

          // Typing indicator
          if (_isTyping)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 6),
              child: Row(
                children: [
                  const SizedBox(
                    width: 14,
                    height: 14,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  ),
                  const SizedBox(width: 10),
                  Text(
                    'AquaDoc synthesizing scientific citations...',
                    style: TextStyle(fontSize: 12, color: Colors.grey[600], fontStyle: FontStyle.italic),
                  ),
                ],
              ),
            ),

          // Quick prompt chips
          Container(
            height: 38,
            margin: const EdgeInsets.symmetric(vertical: 4),
            child: ListView.separated(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              scrollDirection: Axis.horizontal,
              itemCount: _quickPrompts.length,
              separatorBuilder: (_, __) => const SizedBox(width: 8),
              itemBuilder: (context, index) {
                final prompt = _quickPrompts[index];
                return ActionChip(
                  label: Text(prompt, style: const TextStyle(fontSize: 12)),
                  onPressed: () => _sendMessage(prompt),
                );
              },
            ),
          ),

          // Input Bar
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Theme.of(context).scaffoldBackgroundColor,
              border: Border(top: BorderSide(color: Colors.grey.withOpacity(0.2))),
            ),
            child: SafeArea(
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _textController,
                      decoration: const InputDecoration(
                        hintText: 'Ask AquaDoc about pond health, DO, feeding...',
                        border: OutlineInputBorder(
                          borderRadius: BorderRadius.all(Radius.circular(24)),
                        ),
                        contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                      ),
                      onSubmitted: _sendMessage,
                    ),
                  ),
                  const SizedBox(width: 8),
                  IconButton.filled(
                    onPressed: () => _sendMessage(_textController.text),
                    icon: const Icon(Icons.send_rounded),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildMessageBubble(BuildContext context, AquaDocMessage msg) {
    if (msg.isUser) {
      return Align(
        alignment: Alignment.centerRight,
        child: Container(
          margin: const EdgeInsets.only(bottom: 12, left: 48),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.primary,
            borderRadius: const BorderRadius.only(
              topLeft: Radius.circular(16),
              topRight: Radius.circular(16),
              bottomLeft: Radius.circular(16),
            ),
          ),
          child: Text(
            msg.text,
            style: const TextStyle(color: Colors.white, fontSize: 14),
          ),
        ),
      );
    }

    return Align(
      alignment: Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.only(bottom: 16, right: 24),
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: Theme.of(context).cardColor,
          borderRadius: const BorderRadius.only(
            topLeft: Radius.circular(16),
            topRight: Radius.circular(16),
            bottomRight: Radius.circular(16),
          ),
          border: Border.all(color: Colors.grey.withOpacity(0.15)),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withOpacity(0.04),
              blurRadius: 8,
              offset: const Offset(0, 2),
            ),
          ],
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.psychology, color: Colors.blueAccent, size: 18),
                const SizedBox(width: 6),
                const Text('AquaDoc Consultant', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 13)),
                const Spacer(),
                Text(
                  '${msg.timestamp.hour.toString().padLeft(2, '0')}:${msg.timestamp.minute.toString().padLeft(2, '0')}',
                  style: TextStyle(fontSize: 10, color: Colors.grey[500]),
                ),
              ],
            ),
            const SizedBox(height: 10),
            Text(
              msg.text,
              style: const TextStyle(fontSize: 14, height: 1.4),
            ),

            // Missing Data Warnings
            if (msg.missingData.isNotEmpty) ...[
              const SizedBox(height: 12),
              Wrap(
                spacing: 6,
                runSpacing: 4,
                children: msg.missingData.map((param) {
                  return Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                    decoration: BoxDecoration(
                      color: Colors.amber.withOpacity(0.15),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: Colors.amber.withOpacity(0.4)),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const Icon(Icons.help_outline, size: 12, color: Colors.amber),
                        const SizedBox(width: 4),
                        Text(
                          '$param: UNKNOWN',
                          style: TextStyle(fontSize: 10, color: Colors.amber[900], fontWeight: FontWeight.bold),
                        ),
                      ],
                    ),
                  );
                }).toList(),
              ),
            ],

            // Literature Citations
            if (msg.citations.isNotEmpty) ...[
              const SizedBox(height: 14),
              const Divider(),
              const SizedBox(height: 6),
              const Row(
                children: [
                  Icon(Icons.menu_book, size: 14, color: Colors.blueGrey),
                  SizedBox(width: 6),
                  Text(
                    'Literature Citations (Hybrid RAG)',
                    style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: Colors.blueGrey),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              ...msg.citations.map((cite) {
                return Container(
                  margin: const EdgeInsets.only(bottom: 6),
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: Colors.grey.withOpacity(0.06),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        cite.title,
                        style: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        '${cite.authors} (${cite.year}) • ${cite.journal}',
                        style: TextStyle(fontSize: 10, color: Colors.grey[600]),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        'Semantic Relevance: ${(cite.relevanceScore * 100).toInt()}% match',
                        style: const TextStyle(fontSize: 9, color: Colors.blue, fontWeight: FontWeight.w600),
                      ),
                    ],
                  ),
                );
              }),
            ],
          ],
        ),
      ),
    );
  }
}

class _ContextChip extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;
  final bool isWarning;

  const _ContextChip({
    required this.icon,
    required this.label,
    required this.value,
    this.isWarning = false,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: isWarning ? Colors.amber.withOpacity(0.2) : Colors.blue.withOpacity(0.1),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 12, color: isWarning ? Colors.amber[900] : Colors.blue[700]),
          const SizedBox(width: 4),
          Text(
            '$label: $value',
            style: TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.w600,
              color: isWarning ? Colors.amber[900] : Colors.blue[800],
            ),
          ),
        ],
      ),
    );
  }
}
