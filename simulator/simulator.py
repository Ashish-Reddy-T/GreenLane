#!/usr/bin/env python3
"""
GreenLane Fleet Simulator
Spawns virtual EVs that stream telemetry to the Ingestion Service
"""

import grpc
import time
import random
import sys
from concurrent import futures
from datetime import datetime

# Import generated protobuf code
sys.path.insert(0, '../proto')
try:
    import fleet_pb2
    import fleet_pb2_grpc
except ImportError:
    print("❌ Error: Protobuf files not generated!")
    print("   Run: cd .. && make proto && cd simulator")
    print("   Then copy fleet_pb2.py and fleet_pb2_grpc.py to simulator/")
    sys.exit(1)

# Configuration
GRPC_SERVER = "localhost:50051"
API_TOKEN = "greenlane-secret-token"
NUM_CARS = 5
UPDATE_INTERVAL = 2  # seconds

# Manhattan bounding box (approximate)
MANHATTAN_BOUNDS = {
    "lat_min": 40.700,
    "lat_max": 40.850,
    "lon_min": -74.020,
    "lon_max": -73.930,
}


class VirtualCar:
    def __init__(self, car_id):
        self.car_id = car_id
        self.latitude = random.uniform(
            MANHATTAN_BOUNDS["lat_min"], MANHATTAN_BOUNDS["lat_max"]
        )
        self.longitude = random.uniform(
            MANHATTAN_BOUNDS["lon_min"], MANHATTAN_BOUNDS["lon_max"]
        )
        self.battery_level = random.uniform(20.0, 100.0)
        self.velocity = random.uniform(0.0, 60.0)
        self.direction = random.uniform(0, 2 * 3.14159)  # Random direction in radians

    def update_position(self):
        """Simulate car movement with random walk"""
        # Move car slightly in its current direction
        move_distance = 0.001  # ~100 meters
        self.latitude += move_distance * random.uniform(-1, 1)
        self.longitude += move_distance * random.uniform(-1, 1)

        # Keep within Manhattan bounds
        self.latitude = max(
            MANHATTAN_BOUNDS["lat_min"],
            min(MANHATTAN_BOUNDS["lat_max"], self.latitude),
        )
        self.longitude = max(
            MANHATTAN_BOUNDS["lon_min"],
            min(MANHATTAN_BOUNDS["lon_max"], self.longitude),
        )

        # Update battery (drain slightly)
        self.battery_level = max(0.0, self.battery_level - random.uniform(0.1, 0.5))

        # Update velocity
        self.velocity = random.uniform(0.0, 60.0)

    def to_protobuf(self):
        """Convert to protobuf CarStatus message"""
        return fleet_pb2.CarStatus(
            car_id=self.car_id,
            latitude=self.latitude,
            longitude=self.longitude,
            battery_level=self.battery_level,
            velocity=self.velocity,
            timestamp=int(time.time() * 1000),
        )


def simulate_car_stream(car, stub):
    """Simulate a single car's telemetry stream"""
    try:
        # Create metadata with API token
        metadata = [("x-api-token", API_TOKEN)]

        # Create bidirectional stream
        def request_generator():
            while True:
                car.update_position()
                status = car.to_protobuf()
                
                # Color code based on battery level
                if car.battery_level < 20:
                    color = "\033[91m"  # Red
                    indicator = "🔴"
                elif car.battery_level < 50:
                    color = "\033[93m"  # Yellow
                    indicator = "🟡"
                else:
                    color = "\033[92m"  # Green
                    indicator = "🟢"
                
                reset = "\033[0m"
                
                print(
                    f"{indicator} {color}Car {car.car_id}{reset} | "
                    f"Battery: {car.battery_level:5.1f}% | "
                    f"Location: ({car.latitude:.4f}, {car.longitude:.4f}) | "
                    f"Speed: {car.velocity:4.1f} km/h"
                )
                
                yield status
                time.sleep(UPDATE_INTERVAL)

        responses = stub.StreamTelemetry(request_generator(), metadata=metadata)

        # Listen for responses from server
        for response in responses:
            if response.status != fleet_pb2.BOOKING_UNKNOWN:
                print(f"   ↳ Response for {response.car_id}: {response.message}")

    except grpc.RpcError as e:
        print(f"❌ RPC Error for Car {car.car_id}: {e.code()} - {e.details()}")
    except KeyboardInterrupt:
        print(f"\n🛑 Stopping Car {car.car_id}")


def main():
    print("🚗 GreenLane Fleet Simulator")
    print("=" * 60)
    print(f"Spawning {NUM_CARS} virtual cars...")
    print(f"Server: {GRPC_SERVER}")
    print(f"Update interval: {UPDATE_INTERVAL}s")
    print("=" * 60)
    print()

    # Create gRPC channel
    try:
        channel = grpc.insecure_channel(GRPC_SERVER)
        stub = fleet_pb2_grpc.FleetServiceStub(channel)

        # Create virtual cars
        cars = [VirtualCar(f"CAR-{i:03d}") for i in range(1, NUM_CARS + 1)]

        # Run simulations in parallel threads
        with futures.ThreadPoolExecutor(max_workers=NUM_CARS) as executor:
            car_futures = [
                executor.submit(simulate_car_stream, car, stub) for car in cars
            ]

            # Wait for all simulations (runs until Ctrl+C)
            try:
                for future in futures.as_completed(car_futures):
                    future.result()
            except KeyboardInterrupt:
                print("\n\n🛑 Shutting down simulator...")
                executor.shutdown(wait=False)

    except Exception as e:
        print(f"❌ Fatal error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
