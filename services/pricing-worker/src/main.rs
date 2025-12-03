use chrono::Utc;
use log::{error, info};
use rdkafka::config::ClientConfig;
use rdkafka::consumer::{Consumer, StreamConsumer};
use rdkafka::message::Message;
use serde::{Deserialize, Serialize};
use std::time::Duration;
use tokio_postgres::NoTls;

const KAFKA_BROKER: &str = "localhost:19092";
const KAFKA_TOPIC: &str = "fleet-events";
const KAFKA_GROUP_ID: &str = "pricing-worker-group";
const DB_CONNECTION: &str = "host=localhost user=greenlane password=greenlane_password dbname=greenlane";
const MOCK_GRID_URL: &str = "http://localhost:8081/api/pricing";

#[derive(Debug, Deserialize)]
struct TelemetryEvent {
    car_id: String,
    lat: f64,
    lon: f64,
    battery: f64,
    velocity: f64,
    timestamp: i64,
    event_type: String,
}

#[derive(Debug, Deserialize, Serialize)]
struct PriceResponse {
    timestamp: i64,
    price_per_kwh: f64,
    grid_load: String,
    energy_source: String,
    hour: i32,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    env_logger::init();

    info!("🦀 GreenLane Pricing Worker starting...");

    // Connect to PostgreSQL/TimescaleDB
    let (client, connection) = tokio_postgres::connect(DB_CONNECTION, NoTls).await?;

    tokio::spawn(async move {
        if let Err(e) = connection.await {
            error!("PostgreSQL connection error: {}", e);
        }
    });

    info!("✅ Connected to TimescaleDB");

    // Create Kafka consumer
    let consumer: StreamConsumer = ClientConfig::new()
        .set("group.id", KAFKA_GROUP_ID)
        .set("bootstrap.servers", KAFKA_BROKER)
        .set("enable.auto.commit", "true")
        .set("auto.offset.reset", "earliest")
        .create()?;

    consumer.subscribe(&[KAFKA_TOPIC])?;
    info!("✅ Subscribed to Kafka topic: {}", KAFKA_TOPIC);
    info!("📡 Listening for events...");

    // Create HTTP client for fetching prices
    let http_client = reqwest::Client::new();

    // Consume messages
    loop {
        match consumer.recv().await {
            Ok(message) => {
                if let Some(payload) = message.payload() {
                    match std::str::from_utf8(payload) {
                        Ok(json_str) => {
                            if let Ok(event) = serde_json::from_str::<TelemetryEvent>(json_str) {
                                info!(
                                    "📥 Received: Car {} | Battery: {:.1}% | Location: ({:.4}, {:.4})",
                                    event.car_id, event.battery, event.lat, event.lon
                                );

                                // Fetch current pricing from mock grid service
                                match fetch_current_price(&http_client).await {
                                    Ok(price_info) => {
                                        info!(
                                            "💰 Price: ${:.3}/kWh | Load: {} | Source: {}",
                                            price_info.price_per_kwh,
                                            price_info.grid_load,
                                            price_info.energy_source
                                        );

                                        // Write to TimescaleDB (simulate charging session)
                                        if let Err(e) = write_to_timescale(
                                            &client,
                                            &event,
                                            &price_info,
                                        )
                                        .await
                                        {
                                            error!("Failed to write to TimescaleDB: {}", e);
                                        } else {
                                            info!("✅ Written to TimescaleDB");
                                        }
                                    }
                                    Err(e) => {
                                        error!("Failed to fetch pricing: {}", e);
                                    }
                                }
                            }
                        }
                        Err(e) => {
                            error!("Failed to decode message: {}", e);
                        }
                    }
                }
            }
            Err(e) => {
                error!("Kafka error: {}", e);
                tokio::time::sleep(Duration::from_secs(1)).await;
            }
        }
    }
}

async fn fetch_current_price(
    client: &reqwest::Client,
) -> Result<PriceResponse, Box<dyn std::error::Error>> {
    let response = client
        .get(MOCK_GRID_URL)
        .timeout(Duration::from_secs(5))
        .send()
        .await?;

    let price_info = response.json::<PriceResponse>().await?;
    Ok(price_info)
}

async fn write_to_timescale(
    client: &tokio_postgres::Client,
    event: &TelemetryEvent,
    price_info: &PriceResponse,
) -> Result<(), Box<dyn std::error::Error>> {
    // Generate a session_id and station_id for demonstration
    let session_id = format!("session-{}-{}", event.car_id, Utc::now().timestamp());
    let station_id = format!("station-{}", (event.car_id.chars().last().unwrap() as u32) % 10);

    // Simulate energy usage (random between 5-20 kWh)
    let kwh_usage = 10.0 + (event.battery / 10.0);

    // Use DateTime<Utc> instead of NaiveDateTime for TimescaleDB
    let now = Utc::now();

    client
        .execute(
            "INSERT INTO charging_sessions (time, session_id, station_id, car_id, kwh_usage, price_rate)
             VALUES ($1, $2, $3, $4, $5, $6)",
            &[
                &now,
                &session_id,
                &station_id,
                &event.car_id,
                &kwh_usage,
                &price_info.price_per_kwh,
            ],
        )
        .await?;

    Ok(())
}
