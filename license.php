<?php
error_reporting(E_ALL);
ini_set('display_errors', 1);
header('Content-Type: application/json');

// =====================
// CONFIGURAÇÃO MYSQL
// =====================
$DB_HOST = "185.14.92.194";
$DB_USER = "root";
$DB_PASS = "SENHA_AQUI";
$DB_NAME = "nome_database";

// =====================
// CONECTAR MYSQL
// =====================
$conn = new mysqli($DB_HOST, $DB_USER, $DB_PASS, $DB_NAME);
if($conn->connect_error){
    die(json_encode([
        "status" => "error",
        "msg" => "MySQL connection failed",
        "error" => $conn->connect_error
    ]));
}

// =====================
// PEGAR PARAMETROS
// =====================
$action  = isset($_GET['action']) ? $_GET['action'] : null;
$license = isset($_GET['license']) ? $_GET['license'] : null;

if(!$action){
    echo json_encode([
        "status" => "error",
        "msg" => "Missing action"
    ]);
    exit;
}

// ------------------------------
// 1) AUTHENTICATION
// ------------------------------
if($action === "authentication"){

    if(!$license){
        echo json_encode([
            "status" => "error",
            "msg" => "Missing license"
        ]);
        exit;
    }

    $stmt = $conn->prepare("SELECT expiry_date, is_banned FROM licenses WHERE license_key=? LIMIT 1");
    $stmt->bind_param("s", $license);
    $stmt->execute();
    $stmt->store_result();

    if($stmt->num_rows == 0){
        echo json_encode([
            "status" => "invalid",
            "msg" => "License not found"
        ]);
        exit;
    }

    $stmt->bind_result($expiry, $banned);
    $stmt->fetch();

    // Banida
    if($banned == 1){
        echo json_encode([
            "status" => "banned",
            "msg" => "This license is banned"
        ]);
        exit;
    }

    // Expirada
    $now = date("Y-m-d");
    if($now > $expiry){
        echo json_encode([
            "status" => "expired",
            "msg" => "License expired",
            "expiry" => $expiry
        ]);
        exit;
    }

    // Tudo certo
    echo json_encode([
        "status" => "valid",
        "msg" => "License OK",
        "expiry" => $expiry
    ]);
    exit;
}
