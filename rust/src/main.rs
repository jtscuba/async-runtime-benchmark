use std::{collections::HashMap, time::Duration};

use actix_web::{
    web::{self, Data},
    App, FromRequest, HttpRequest, HttpResponse, HttpServer, Responder,
};
use aws_config::BehaviorVersion;
use aws_sdk_dynamodb::{
    client::Client, config::{Credentials, Region, SharedCredentialsProvider}, error::SdkError, operation::{put_item::{PutItemError, PutItemOutput}, scan::{ScanError, ScanOutput}}, types::{AttributeDefinition, AttributeValue, KeySchemaElement, KeyType, ScalarAttributeType}
};
use aws_smithy_runtime::client::http::hyper_014::HyperClientBuilder;
use aws_smithy_runtime_api::http::Response;
use aws_smithy_types::body::SdkBody;
use hyper_rustls::ConfigBuilderExt;
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
struct User {
    name: String,
    email: String,
    attributes: HashMap<String, String>,
}

#[derive(Debug, Serialize, Deserialize)]
struct DbUser {
    id: String,
    name: String,
    email: String,
    attributes: HashMap<String, String>,
}

#[derive(Debug, Serialize, Deserialize)]
struct ListResponse {
    items: Vec<DbUser>,
    continuation_token: Option<String>,
}

async fn create_user(client: Data<Client>, req: web::Json<User>) -> impl Responder {
    let table_name = "MyTable";
    let key = uuid::Uuid::new_v4().to_string();

    let mut result = create_user_impl(client.clone(), table_name, &key, &req).await;
    for _ in 0..4 {
        if result.is_ok() {
            break;
        }
        result = create_user_impl(client.clone(), table_name, &key, &req).await;
    }

    result.unwrap();

    HttpResponse::Created()
}

async fn create_user_impl(client: Data<Client>, table_name: &str, key: &String, req: &web::Json<User>) -> Result<PutItemOutput, SdkError<PutItemError, Response<SdkBody>>> {
    let result: Result<PutItemOutput, SdkError<PutItemError, Response<SdkBody>>> = client
        .put_item()
        .table_name(table_name)
        .item("Id", AttributeValue::S(key.clone()))
        .item("Name", AttributeValue::S(req.name.clone()))
        .item("Email", AttributeValue::S(req.email.clone()))
        .item(
            "Attributes",
            AttributeValue::M(
                req.attributes
                    .clone()
                    .into_iter()
                    .map(|(k, v)| (k, AttributeValue::S(v)))
                    .collect(),
            ),
        )
        .send()
        .await;
    result
}

async fn list_users(client: Data<Client>, req: HttpRequest) -> impl Responder {
    let table_name: &str = "MyTable";
    let limit = 1000;
    let continuation_token = web::Query::<String>::extract(&req)
        .await
        .ok()
        .map(|v| v.into_inner());

    let mut result = list_users_impl(client.clone(), table_name, limit, &continuation_token).await;
    for _ in 0..4 {
        if result.is_ok() {
            break;
        }
        result = list_users_impl(client.clone(), table_name, limit, &continuation_token).await;
    }

    let result = result.unwrap();

    let list_resp = ListResponse {
        items: result
            .items
            .unwrap()
            .into_iter()
            .map(|item| DbUser {
                id: item.get("Id").unwrap().as_s().unwrap().to_string(), // .s.clone().unwrap(),
                name: item.get("Name").unwrap().as_s().unwrap().to_string(),
                email: item.get("Email").unwrap().as_s().unwrap().to_string(),
                attributes: item
                    .get("Attributes")
                    .unwrap()
                    .as_m()
                    .clone()
                    .unwrap()
                    .into_iter()
                    .map(|(k, v)| (k.clone(), v.as_s().unwrap().clone()))
                    .collect(),
            })
            .collect(),
        continuation_token: result
            .last_evaluated_key
            .and_then(|key| key.get("Id").unwrap().as_s().ok().cloned()),
    };

    HttpResponse::Ok().json(list_resp)
}

async fn list_users_impl(client: Data<Client>, table_name: &str, limit: i32, continuation_token: &Option<String>) -> Result<ScanOutput, SdkError<ScanError, Response>> {
    let mut input = client.scan().table_name(table_name).limit(limit);

    if let Some(token) = continuation_token {
        input = input.exclusive_start_key("Id", AttributeValue::S(token.to_owned()));
    }
    let result: Result<ScanOutput, SdkError<ScanError, Response>> = input.send().await;
    result
}

#[tokio::main(flavor = "multi_thread")]
async fn main() -> std::io::Result<()> {
    env_logger::init();

    let mut builder: aws_sdk_dynamodb::config::Builder = aws_sdk_dynamodb::Config::builder();
    builder.set_behavior_version(Some(BehaviorVersion::v2023_11_09()));
    builder.set_region(Some(Region::from_static("local")));
    builder.set_endpoint_url(Some("http://localhost:8000".to_string()));
    builder.set_credentials_provider(Some(SharedCredentialsProvider::new(Credentials::new(
        "AKID",
        "SECRET_KEY",
        Some("TOKEN".to_string()),
        None,
        "test",
    ))));

    let https_connector = hyper_rustls::HttpsConnectorBuilder::new()
        .with_tls_config(
         rustls::ClientConfig::builder()
             .with_cipher_suites(&[
                 // TLS1.3 suites
                 rustls::cipher_suite::TLS13_AES_256_GCM_SHA384,
                 rustls::cipher_suite::TLS13_AES_128_GCM_SHA256,
                 // TLS1.2 suites
                 rustls::cipher_suite::TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
                 rustls::cipher_suite::TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
                 rustls::cipher_suite::TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
                 rustls::cipher_suite::TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
                 rustls::cipher_suite::TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
             ])
             .with_safe_default_kx_groups()
             .with_safe_default_protocol_versions()
             .expect("Error with the TLS configuration. Please file a bug report under https://github.com/smithy-lang/smithy-rs/issues.")
             .with_native_roots()
             .with_no_client_auth()
     )
     .https_or_http()
     .enable_http1()
     .enable_http2()
     .build();

    let mut hyper_builder = hyper::Client::builder();
    hyper_builder
        .http2_keep_alive_timeout(Duration::from_secs(20))
        .http2_keep_alive_interval(Duration::from_secs(10))
        .pool_idle_timeout(None)
        .pool_max_idle_per_host(1000);
        // .http2_max_concurrent_reset_streams(1000);

    let hyper_client = HyperClientBuilder::new()
        .hyper_builder(hyper_builder)
        .build(https_connector);
    builder.set_http_client(Some(hyper_client));

    // let retry_config = RetryConfig::standard().with_max_attempts(10);
    // builder.set_retry_config(Some(retry_config));

    let config = builder.build();
    let client = Client::from_conf(config);
    let table_name = "MyTable";

    // delete the table if it exists, then recreate it
    let _result = client.delete_table().table_name(table_name).send().await;

    println!("deleted table {}", table_name);

    client
        .create_table()
        .table_name(table_name)
        .attribute_definitions(
            AttributeDefinition::builder()
                .attribute_name("Id")
                .attribute_type(ScalarAttributeType::S)
                .build()
                .unwrap(),
        )
        .key_schema(
            KeySchemaElement::builder()
                .attribute_name("Id")
                .key_type(KeyType::Hash)
                .build()
                .unwrap(),
        )
        .provisioned_throughput(
            aws_sdk_dynamodb::types::ProvisionedThroughput::builder()
                .read_capacity_units(5000)
                .write_capacity_units(5000)
                .build()
                .unwrap(),
        )
        .send()
        .await
        .unwrap();

    println!("created table {}", table_name);
    HttpServer::new(move || {
        let app = App::new()
            .app_data(Data::new(client.clone()))
            .service(web::resource("/create").route(web::post().to(create_user)))
            .service(web::resource("/list").route(web::get().to(list_users)))
            .wrap(actix_web::middleware::Logger::default());

        app
    })
    // .workers(6)
    .bind("localhost:8081")?
    .run()
    .await
}
