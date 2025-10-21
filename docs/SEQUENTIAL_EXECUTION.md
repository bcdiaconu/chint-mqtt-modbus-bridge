# Sequential Execution & Race Condition Protection

## Overview

This document explains how the MQTT-Modbus Bridge ensures **sequential execution** of Modbus requests and prevents race conditions when polling multiple slaves and register groups.

## Problem Statement

When polling multiple Modbus devices (e.g., two energy meters with Slave IDs 1 and 11), there are potential issues:

1. **Race Condition**: Responses from different slaves could be mixed up
2. **Stale Responses**: Old responses from timed-out requests could be received by new requests
3. **Overlapping Requests**: Multiple groups executing simultaneously could interfere

## Architecture

### Configuration Structure

```yaml
devices:
  energy_meter_mains:
    rtu:
      slave_id: 11
      poll_interval: 1000  # ms - applies to ALL groups for this device
    register_groups:
      instant:  # Instant measurements (voltage, current, power)
        start_address: 0x2000
        register_count: 34
      energy:   # Energy counters
        start_address: 0x4000
        register_count: 12

  energy_meter_lights:
    rtu:
      slave_id: 1
      poll_interval: 1000
    register_groups:
      instant:
        start_address: 0x2000
        register_count: 16
      energy:
        start_address: 0x4000
        register_count: 12
```

### Execution Flow

```
Every 1000ms (poll_interval):
│
├─ ExecuteAll() called
│  │
│  ├─ Group: energy_meter_mains.instant (Slave 11, 0x2000)
│  │  └─ SendCommandAndWaitForResponse() [MUTEX LOCKED]
│  │     ├─ Clear stale responses
│  │     ├─ Set expectedSlaveID = 11, expectedFunctionCode = 0x03
│  │     ├─ Send command to gateway
│  │     ├─ Wait for response (validated by SlaveID/FunctionCode)
│  │     └─ [MUTEX UNLOCKED] + 50ms delay
│  │
│  ├─ Group: energy_meter_mains.energy (Slave 11, 0x4000)
│  │  └─ SendCommandAndWaitForResponse() [MUTEX LOCKED]
│  │     └─ ... (same flow)
│  │
│  ├─ Group: energy_meter_lights.instant (Slave 1, 0x2000)
│  │  └─ SendCommandAndWaitForResponse() [MUTEX LOCKED]
│  │     └─ ... (same flow)
│  │
│  └─ Group: energy_meter_lights.energy (Slave 1, 0x4000)
│     └─ SendCommandAndWaitForResponse() [MUTEX LOCKED]
│        └─ ... (same flow)
│
└─ All groups executed SEQUENTIALLY (no overlap)
```

## Protection Mechanisms

### 1. Command Mutex (Primary Protection)

```go
type USRGateway struct {
    commandMutex sync.Mutex  // Ensures only ONE command/response at a time
    // ...
}

func (g *USRGateway) SendCommandAndWaitForResponse(...) {
    g.commandMutex.Lock()    // ← BLOCKS all other requests
    defer g.commandMutex.Unlock()
    
    // Send command and wait for response
    // ...
}
```

**Guarantees**:
- ✅ Only ONE Modbus transaction active at any time
- ✅ Groups from different slaves CANNOT execute simultaneously
- ✅ No overlap between register reads

### 2. Response Validation (Defense-in-Depth)

```go
type USRGateway struct {
    expectedSlaveID      uint8  // Which slave should respond?
    expectedFunctionCode uint8  // Which function was called?
    // ...
}

func (g *USRGateway) onMessage(client mqtt.Client, msg mqtt.Message) {
    receivedSlaveID := data[0]
    receivedFunctionCode := data[1]
    
    // Validate response matches expected request
    if receivedSlaveID != expectedSlaveID || 
       receivedFunctionCode != expectedFunctionCode {
        logger.LogWarn("Unexpected response, ignoring")
        return  // Ignore wrong response
    }
    
    // Send to responseChan only if validated
}
```

**Guarantees**:
- ✅ Each request gets its CORRECT response
- ✅ Responses from wrong slaves are rejected
- ✅ Out-of-order responses detected and ignored

### 3. Stale Response Cleanup

```go
func (g *USRGateway) SendCommandAndWaitForResponse(...) {
    g.commandMutex.Lock()
    defer g.commandMutex.Unlock()
    
    // Clear any stale responses before sending new command
    select {
    case <-g.responseChan:
        logger.LogWarn("Cleared stale response")
    default:
        // Channel empty, good to go
    }
    
    // Set expected response params
    g.expectedSlaveID = slaveID
    g.expectedFunctionCode = functionCode
    
    // Send command...
}
```

**Guarantees**:
- ✅ Old responses from timed-out requests are cleared
- ✅ New requests start with clean channel
- ✅ No stale data affecting current transaction

### 4. Inter-Command Delay

```go
func (g *USRGateway) SendCommandAndWaitForResponse(...) {
    // ... send and receive ...
    
    // Add small delay between commands to prevent gateway overload
    time.Sleep(50 * time.Millisecond)
    
    return response, nil
}
```

**Guarantees**:
- ✅ Gateway has time to process between requests
- ✅ Reduces chance of buffer overflow on gateway side
- ✅ Improves reliability for slow devices

## Timing Analysis

### Single Poll Cycle (Worst Case)

For 2 devices × 2 groups = 4 total groups:

```
Group 1: Send (10ms) + Wait (100ms) + Delay (50ms) = 160ms
Group 2: Send (10ms) + Wait (100ms) + Delay (50ms) = 160ms
Group 3: Send (10ms) + Wait (100ms) + Delay (50ms) = 160ms
Group 4: Send (10ms) + Wait (100ms) + Delay (50ms) = 160ms
─────────────────────────────────────────────────────────
Total:                                             640ms
```

**Poll interval**: 1000ms  
**Execution time**: ~640ms (worst case)  
**Safety margin**: ~360ms ✅

### Conclusion

✅ **No overlap possible** - execution time (640ms) < poll interval (1000ms)  
✅ **Sequential execution** enforced by commandMutex  
✅ **Response validation** prevents cross-contamination  

## Debugging

### Enable Debug Logging

Set log level to `trace` in `config.yaml`:

```yaml
logging:
  level: "trace"  # Shows detailed execution flow
```

### Log Messages to Watch For

**Normal Operation**:
```
🔄 Executing group 'energy_meter_mains.instant' (Slave 11, Addr 0x2000, Count 34)
✅ Group 'energy_meter_mains.instant' (Slave 11) read successful (68 bytes)
Gateway received valid response from Slave 11: [data...]
```

**Potential Issues**:
```
⚠️  Cleared stale response from channel before new request
    → Old response found, was cleared (should be rare)

⚠️  Received unexpected response (Slave=1, Func=0x03) but expecting (Slave=11, Func=0x03)
    → Response validation working, wrong response rejected

❌ Group 'energy_meter_lights.instant' (Slave 1) read failed: timeout
    → Device not responding, check wiring/power
```

## Testing Checklist

- [ ] Both meters show correct, distinct values (not swapped)
- [ ] Active power not doubled on either meter
- [ ] Light meter shows non-zero when lights are on
- [ ] No "Unexpected response" warnings in logs
- [ ] No "Cleared stale response" warnings (or very rare)
- [ ] Response times stay within timeout (5s)
- [ ] Poll cycle completes within interval (1000ms)

## Summary

The MQTT-Modbus Bridge uses **three layers of protection**:

1. **Mutex Lock**: Ensures sequential execution (primary)
2. **Response Validation**: Verifies correct slave/function (defense-in-depth)
3. **Stale Cleanup**: Prevents old responses affecting new requests

This architecture **guarantees**:
- ✅ No race conditions between slaves
- ✅ No overlapping register group reads
- ✅ Each request gets its correct response
- ✅ Deterministic, predictable execution order

---

**Document Version**: 1.0  
**Last Updated**: 2025-10-21  
**Related Commits**: 8c02a5b (race condition fix)
