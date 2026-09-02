import 'dart:async';
import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:flutter_blue_plus/flutter_blue_plus.dart';
import 'package:logger/logger.dart';
import 'package:pointycastle/export.dart';

enum BleConnectionState {
  disconnected,
  scanning,
  connecting,
  connected,
  provisioning,
}

class BleDevice {
  final String id;
  final String name;
  final int rssi;
  final BluetoothDevice? device;

  BleDevice({
    required this.id,
    required this.name,
    required this.rssi,
    this.device,
  });
}

/// ECDH Key Exchange for secure BLE provisioning
class ECDHKeyExchange {
  late AsymmetricKeyPair<PublicKey, PrivateKey> _keyPair;
  Uint8List? _sharedSecret;

  ECDHKeyExchange() {
    _generateKeyPair();
  }

  void _generateKeyPair() {
    final keyParams = ECKeyGeneratorParameters(ECCurve_secp256r1());
    final random = FortunaRandom();
    random.seed(
      KeyParameter(
        Uint8List.fromList(
          List.generate(32, (_) => Random.secure().nextInt(256)),
        ),
      ),
    );

    final generator =
        ECKeyGenerator()..init(ParametersWithRandom(keyParams, random));

    _keyPair = generator.generateKeyPair();
  }

  /// Get our public key to send to the device
  Uint8List getPublicKey() {
    final publicKey = _keyPair.publicKey as ECPublicKey;
    final q = publicKey.Q!;
    // Encode as uncompressed point (0x04 || x || y)
    final x = _bigIntToBytes(q.x!.toBigInteger()!, 32);
    final y = _bigIntToBytes(q.y!.toBigInteger()!, 32);
    return Uint8List.fromList([0x04, ...x, ...y]);
  }

  /// Compute shared secret from device's public key
  Uint8List computeSharedSecret(Uint8List devicePublicKeyBytes) {
    // Parse device public key (uncompressed format)
    if (devicePublicKeyBytes[0] != 0x04 || devicePublicKeyBytes.length != 65) {
      throw ArgumentError('Invalid public key format');
    }

    final x = _bytesToBigInt(devicePublicKeyBytes.sublist(1, 33));
    final y = _bytesToBigInt(devicePublicKeyBytes.sublist(33, 65));

    final curve = ECCurve_secp256r1();
    final devicePublicKey = ECPublicKey(curve.curve.createPoint(x, y), curve);

    // Compute ECDH shared secret
    final privateKey = _keyPair.privateKey as ECPrivateKey;
    final sharedPoint = devicePublicKey.Q! * privateKey.d;

    _sharedSecret = _bigIntToBytes(sharedPoint!.x!.toBigInteger()!, 32);
    return _sharedSecret!;
  }

  /// Encrypt data using the shared secret (AES-256-GCM)
  Uint8List encrypt(Uint8List plaintext) {
    if (_sharedSecret == null) {
      throw StateError('Shared secret not computed');
    }

    final random = FortunaRandom();
    random.seed(
      KeyParameter(
        Uint8List.fromList(
          List.generate(32, (_) => Random.secure().nextInt(256)),
        ),
      ),
    );

    // Generate random IV
    final iv = Uint8List(12);
    for (var i = 0; i < 12; i++) {
      iv[i] = random.nextUint8();
    }

    // AES-GCM encryption
    final cipher = GCMBlockCipher(AESEngine())..init(
      true,
      AEADParameters(KeyParameter(_sharedSecret!), 128, iv, Uint8List(0)),
    );

    final ciphertext = cipher.process(plaintext);

    // Return IV + ciphertext
    return Uint8List.fromList([...iv, ...ciphertext]);
  }

  /// Decrypt data using the shared secret
  Uint8List decrypt(Uint8List ciphertext) {
    if (_sharedSecret == null) {
      throw StateError('Shared secret not computed');
    }

    // Extract IV and ciphertext
    final iv = ciphertext.sublist(0, 12);
    final encrypted = ciphertext.sublist(12);

    // AES-GCM decryption
    final cipher = GCMBlockCipher(AESEngine())..init(
      false,
      AEADParameters(KeyParameter(_sharedSecret!), 128, iv, Uint8List(0)),
    );

    return cipher.process(encrypted);
  }

  Uint8List _bigIntToBytes(BigInt number, int length) {
    final bytes = Uint8List(length);
    var temp = number;
    for (var i = length - 1; i >= 0; i--) {
      bytes[i] = (temp & BigInt.from(0xff)).toInt();
      temp = temp >> 8;
    }
    return bytes;
  }

  BigInt _bytesToBigInt(Uint8List bytes) {
    var result = BigInt.zero;
    for (var byte in bytes) {
      result = (result << 8) | BigInt.from(byte);
    }
    return result;
  }
}

class BleService {
  static final BleService _instance = BleService._internal();
  factory BleService() => _instance;
  BleService._internal();

  final Logger _logger = Logger();

  // Service and characteristic UUIDs for Smart Fish Feeder
  static const String serviceUuid = '12345678-1234-5678-1234-56789abcdef0';
  static const String wifiConfigCharUuid =
      '12345678-1234-5678-1234-56789abcdef1';
  static const String cellularConfigCharUuid =
      '12345678-1234-5678-1234-56789abcdef2';
  static const String deviceInfoCharUuid =
      '12345678-1234-5678-1234-56789abcdef3';
  static const String provisionStatusCharUuid =
      '12345678-1234-5678-1234-56789abcdef4';
  static const String bindingCodeCharUuid =
      '12345678-1234-5678-1234-56789abcdef5';

  BluetoothDevice? _connectedDevice;
  BluetoothCharacteristic? _wifiConfigChar;
  BluetoothCharacteristic? _cellularConfigChar;
  BluetoothCharacteristic? _deviceInfoChar;
  BluetoothCharacteristic? _provisionStatusChar;
  BluetoothCharacteristic? _bindingCodeChar;

  ECDHKeyExchange? _ecdhKeyExchange;

  final _stateController = StreamController<BleConnectionState>.broadcast();
  Stream<BleConnectionState> get connectionState => _stateController.stream;

  final _devicesController = StreamController<List<BleDevice>>.broadcast();
  Stream<List<BleDevice>> get discoveredDevices => _devicesController.stream;

  BleConnectionState _currentState = BleConnectionState.disconnected;
  BleConnectionState get currentState => _currentState;

  List<BleDevice> _scanResults = [];
  StreamSubscription? _scanResultsSubscription;

  Future<bool> isBluetoothAvailable() async {
    return await FlutterBluePlus.isSupported;
  }

  Future<bool> isBluetoothOn() async {
    final state = await FlutterBluePlus.adapterState.first;
    return state == BluetoothAdapterState.on;
  }

  Future<void> requestBluetoothOn() async {
    await FlutterBluePlus.turnOn();
  }

  Future<void> startScan({
    Duration timeout = const Duration(seconds: 10),
  }) async {
    if (!await isBluetoothOn()) {
      _logger.w('Bluetooth is not enabled');
      await requestBluetoothOn();
      return;
    }

    _updateState(BleConnectionState.scanning);
    _scanResults.clear();

    await _scanResultsSubscription?.cancel();
    _scanResultsSubscription = FlutterBluePlus.scanResults.listen((results) {
      _scanResults =
          results
              .where((r) {
                final name = r.device.platformName;
                return name.contains('SmartFishFeeder') ||
                    name.contains('SFF-') ||
                    r.advertisementData.serviceUuids.contains(
                      Guid(serviceUuid),
                    );
              })
              .map(
                (r) => BleDevice(
                  id: r.device.remoteId.str,
                  name:
                      r.device.platformName.isNotEmpty
                          ? r.device.platformName
                          : 'Unknown Device',
                  rssi: r.rssi,
                  device: r.device,
                ),
              )
              .toList();
      _devicesController.add(_scanResults);
    });

    await FlutterBluePlus.startScan(timeout: timeout);
    await Future.delayed(timeout);
    _updateState(BleConnectionState.disconnected);
  }

  Future<void> stopScan() async {
    await _scanResultsSubscription?.cancel();
    _scanResultsSubscription = null;
    await FlutterBluePlus.stopScan();
    _updateState(BleConnectionState.disconnected);
  }

  Future<bool> connectToDevice(String deviceId) async {
    try {
      _updateState(BleConnectionState.connecting);

      // Find the device from scan results
      final bleDevice = _scanResults.firstWhere(
        (d) => d.id == deviceId,
        orElse: () => throw Exception('Device not found'),
      );

      if (bleDevice.device == null) {
        throw Exception('Invalid device');
      }

      await bleDevice.device!.connect(autoConnect: false);
      _connectedDevice = bleDevice.device;

      // Discover services
      final services = await _connectedDevice!.discoverServices();

      for (final service in services) {
        if (service.uuid.toString().toLowerCase() ==
            serviceUuid.toLowerCase()) {
          for (final char in service.characteristics) {
            final charUuid = char.uuid.toString().toLowerCase();
            if (charUuid == wifiConfigCharUuid.toLowerCase()) {
              _wifiConfigChar = char;
            } else if (charUuid == cellularConfigCharUuid.toLowerCase()) {
              _cellularConfigChar = char;
            } else if (charUuid == deviceInfoCharUuid.toLowerCase()) {
              _deviceInfoChar = char;
            } else if (charUuid == provisionStatusCharUuid.toLowerCase()) {
              _provisionStatusChar = char;
            } else if (charUuid == bindingCodeCharUuid.toLowerCase()) {
              _bindingCodeChar = char;
            }
          }
        }
      }

      _updateState(BleConnectionState.connected);
      _logger.i('Connected to ${bleDevice.name}');
      return true;
    } catch (e) {
      _logger.e('Connection failed: $e');
      _updateState(BleConnectionState.disconnected);
      return false;
    }
  }

  Future<void> disconnect() async {
    await _connectedDevice?.disconnect();
    _connectedDevice = null;
    _wifiConfigChar = null;
    _cellularConfigChar = null;
    _deviceInfoChar = null;
    _provisionStatusChar = null;
    _bindingCodeChar = null;
    _updateState(BleConnectionState.disconnected);
  }

  Future<DeviceInfo?> getDeviceInfo() async {
    if (_deviceInfoChar == null) return null;

    try {
      final data = await _deviceInfoChar!.read();
      final json = jsonDecode(utf8.decode(data));
      return DeviceInfo.fromJson(json);
    } catch (e) {
      _logger.e('Failed to read device info: $e');
      return null;
    }
  }

  Future<bool> provisionWifi(String ssid, String password) async {
    if (_wifiConfigChar == null) {
      _logger.e('WiFi config characteristic not found');
      return false;
    }

    try {
      _updateState(BleConnectionState.provisioning);

      final config = jsonEncode({
        'type': 'wifi',
        'ssid': ssid,
        'password': password,
        'timestamp': DateTime.now().millisecondsSinceEpoch,
      });

      // Use ECDH encryption if key exchange was performed
      Uint8List dataToSend;
      if (_ecdhKeyExchange != null) {
        dataToSend = _ecdhKeyExchange!.encrypt(utf8.encode(config));
      } else {
        dataToSend = utf8.encode(config);
      }

      await _wifiConfigChar!.write(dataToSend, withoutResponse: false);

      // Wait for provisioning to complete
      await _waitForProvisioningComplete();

      _updateState(BleConnectionState.connected);
      return true;
    } catch (e) {
      _logger.e('WiFi provisioning failed: $e');
      _updateState(BleConnectionState.connected);
      return false;
    }
  }

  Future<bool> provisionCellular(
    String apn, {
    String? username,
    String? password,
  }) async {
    if (_cellularConfigChar == null) {
      _logger.e('Cellular config characteristic not found');
      return false;
    }

    try {
      _updateState(BleConnectionState.provisioning);

      final config = jsonEncode({
        'type': 'cellular',
        'apn': apn,
        'username': username ?? '',
        'password': password ?? '',
        'timestamp': DateTime.now().millisecondsSinceEpoch,
      });

      // Use ECDH encryption if key exchange was performed
      Uint8List dataToSend;
      if (_ecdhKeyExchange != null) {
        dataToSend = _ecdhKeyExchange!.encrypt(utf8.encode(config));
      } else {
        dataToSend = utf8.encode(config);
      }

      await _cellularConfigChar!.write(dataToSend, withoutResponse: false);

      // Wait for provisioning to complete
      await _waitForProvisioningComplete();

      _updateState(BleConnectionState.connected);
      return true;
    } catch (e) {
      _logger.e('Cellular provisioning failed: $e');
      _updateState(BleConnectionState.connected);
      return false;
    }
  }

  /// Perform ECDH key exchange with the device for secure provisioning
  Future<bool> performKeyExchange() async {
    if (_connectedDevice == null) {
      _logger.e('No device connected');
      return false;
    }

    try {
      _ecdhKeyExchange = ECDHKeyExchange();

      // Get our public key
      final ourPublicKey = _ecdhKeyExchange!.getPublicKey();
      _logger.d('Our public key: ${ourPublicKey.length} bytes');

      // Find key exchange characteristic
      final services = await _connectedDevice!.discoverServices();
      BluetoothCharacteristic? keyExchangeChar;

      for (final service in services) {
        if (service.uuid.toString().toLowerCase() ==
            serviceUuid.toLowerCase()) {
          for (final char in service.characteristics) {
            // Key exchange characteristic UUID
            if (char.uuid.toString().toLowerCase() ==
                '12345678-1234-5678-1234-56789abcdef6') {
              keyExchangeChar = char;
              break;
            }
          }
        }
      }

      if (keyExchangeChar == null) {
        _logger.w(
          'Key exchange characteristic not found, using unencrypted provisioning',
        );
        _ecdhKeyExchange = null;
        return false;
      }

      // Send our public key
      await keyExchangeChar.write(ourPublicKey, withoutResponse: false);

      // Read device's public key
      final devicePublicKey = await keyExchangeChar.read();

      if (devicePublicKey.isEmpty) {
        _logger.e('Failed to receive device public key');
        _ecdhKeyExchange = null;
        return false;
      }

      // Compute shared secret
      _ecdhKeyExchange!.computeSharedSecret(
        Uint8List.fromList(devicePublicKey),
      );
      _logger.i('ECDH key exchange completed successfully');

      return true;
    } catch (e) {
      _logger.e('Key exchange failed: $e');
      _ecdhKeyExchange = null;
      return false;
    }
  }

  Future<void> _waitForProvisioningComplete() async {
    if (_provisionStatusChar == null) {
      // If no status characteristic, just wait a bit
      await Future.delayed(const Duration(seconds: 5));
      return;
    }

    // Subscribe to status notifications
    await _provisionStatusChar!.setNotifyValue(true);

    final completer = Completer<void>();
    late StreamSubscription subscription;

    subscription = _provisionStatusChar!.onValueReceived.listen((data) {
      final status = utf8.decode(data);
      _logger.d('Provisioning status: $status');

      if (status == 'complete' || status == 'success') {
        subscription.cancel();
        completer.complete();
      } else if (status == 'error' || status == 'failed') {
        subscription.cancel();
        completer.completeError(Exception('Provisioning failed'));
      }
    });

    // Timeout after 30 seconds
    await completer.future.timeout(
      const Duration(seconds: 30),
      onTimeout: () {
        subscription.cancel();
        throw TimeoutException('Provisioning timed out');
      },
    );
  }

  Future<String?> getBindingCode() async {
    if (_bindingCodeChar == null) {
      _logger.e('Binding code characteristic not found');
      return null;
    }

    try {
      final data = await _bindingCodeChar!.read();
      final code = utf8.decode(data).trim();
      _logger.d('Binding code: $code');
      return code.isNotEmpty ? code : null;
    } catch (e) {
      _logger.e('Failed to read binding code: $e');
      return null;
    }
  }

  Future<bool> sendCommand(String command, Map<String, dynamic> payload) async {
    if (_connectedDevice == null) return false;

    try {
      final commandData = jsonEncode({
        'command': command,
        'payload': payload,
        'timestamp': DateTime.now().millisecondsSinceEpoch,
      });

      // Find command characteristic and write
      // This would need a command characteristic UUID
      _logger.d('Sending BLE command: $command - $commandData');
      return true;
    } catch (e) {
      _logger.e('Failed to send command: $e');
      return false;
    }
  }

  void _updateState(BleConnectionState state) {
    _currentState = state;
    _stateController.add(state);
  }

  void dispose() {
    _scanResultsSubscription?.cancel();
    disconnect();
    _stateController.close();
    _devicesController.close();
  }
}

class DeviceInfo {
  final String deviceId;
  final String serialNumber;
  final String firmwareVersion;
  final String hardwareVersion;
  final String macAddress;
  final bool isProvisioned;

  DeviceInfo({
    required this.deviceId,
    required this.serialNumber,
    required this.firmwareVersion,
    required this.hardwareVersion,
    required this.macAddress,
    required this.isProvisioned,
  });

  factory DeviceInfo.fromJson(Map<String, dynamic> json) {
    return DeviceInfo(
      deviceId: json['device_id'] ?? '',
      serialNumber: json['serial_number'] ?? '',
      firmwareVersion: json['firmware_version'] ?? '',
      hardwareVersion: json['hardware_version'] ?? '',
      macAddress: json['mac_address'] ?? '',
      isProvisioned: json['is_provisioned'] ?? false,
    );
  }
}
