# THC Control System

A privacy-focused THC dosage calculator built for harm reduction.  
The system estimates optimal cannabis dosage based on user physiology, tolerance, strain strength, and consumption method.

## Features

### Physiological Modeling
Calculates baseline THC requirements using:
- Weight
- Height
- Gender

### Tolerance Tracking
Adjusts dosage recommendations dynamically based on previous sessions and usage frequency.

### Consumption Method Accuracy
Supports multiple intake methods with bioavailability correction:
- Vaporizer
- Bong
- Joint
- Edibles

### Automatic Database Initialization
The SQLite database (`system.db`) is created automatically on first launch.  
No manual setup or configuration required.

### Dual Interface
Use either:
- Web Interface (`http://localhost:8080`)
- Command Line Interface (CLI)

### Privacy-Focused Build
Compiled as a static binary with stripped metadata:
- `-trimpath`
- `-ldflags="-s -w"`

---

## Installation

### Download Binary

Download the latest release for your operating system:

`https://github.com/trusree/THC-Control-System/releases`
