#!/bin/bash
# Run the Fleet Simulator
# Make sure all other services are running first!

cd /Users/AshishR_T/Desktop/temp/greenLane/simulator

echo "🚗 Starting Fleet Simulator..."
echo ""
echo "Expected services running:"
echo "  ✓ Mock Grid (Terminal 1)"
echo "  ✓ Ingestion Service (Terminal 2)" 
echo "  ✓ Pricing Worker (Terminal 3)"
echo "  ✓ Live Ops CLI (Terminal 4)"
echo ""
echo "Starting in 3 seconds..."
sleep 3

python3 simulator.py
