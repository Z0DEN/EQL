package com.example.eqlapp

import eqlcore.Eqlcore
import android.os.Bundle
import android.util.Log
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import androidx.lifecycle.lifecycleScope
import androidx.compose.ui.text.input.KeyboardType

class MainActivity : ComponentActivity(), eqlcore.MessageReceiver{
    private var messages by mutableStateOf(listOf<String>())
    private var nodeId by mutableStateOf("Нода не запущена")
    private var nodeInfo by mutableStateOf("")
    private var messageText by mutableStateOf("")

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Регистрируем себя как receiver для сообщений
        Eqlcore.setMessageReceiver(this)

        setContent {
            NodeScreen(
                nodeId = nodeId,
                nodeInfo = nodeInfo,
                messages = messages,
                messageText = messageText,
                onMessageTextChange = { newText -> messageText = newText },
                onStartNode = { startNode() },
                onSendMessage = {
                    if (messageText.isNotBlank()) {
                        sendMessage(messageText)
                        messageText = "" // Очищаем поле после отправки
                    }
                },
                scope = lifecycleScope
            )
        }
    }

    override fun onMessageReceived(from: String, msg: String) {
        Log.e("ChatNode", "Получено сообщение от $from: $msg")

        // Обновляем UI в главном потоке
        runOnUiThread {
            messages = messages + "От $from: $msg"
        }
    }

    private fun startNode() {
        lifecycleScope.launch(Dispatchers.IO) {
            try {
                val id = Eqlcore.startNode("/ip4/0.0.0.0/tcp/0/ws", "chat-room")
                nodeId = "Node ID: $id"
                Log.e("ChatNode", nodeId)

                // Получаем информацию о ноде
                val info = Eqlcore.getNodeInfo()
                nodeInfo = "Node Info: $info"
                Log.e("ChatNode", "Node Info: $info")

            } catch (e: Exception) {
                Log.e("ChatNode", "Ошибка запуска ноды: ${e.message}")
                nodeId = "Ошибка: ${e.message}"
            }
        }
    }

    private fun sendMessage(message: String) {
        lifecycleScope.launch(Dispatchers.IO) {
            try {
                val result = Eqlcore.sendMessage(message)
                Log.e("ChatNode", "Результат отправки: $result")

                // Добавляем своё сообщение в список
                runOnUiThread {
                    messages = messages + "Я: $message"
                }
            } catch (e: Exception) {
                Log.e("ChatNode", "Ошибка отправки сообщения: ${e.message}")
            }
        }
    }
}

@Composable
fun NodeScreen(
    nodeId: String,
    nodeInfo: String,
    messages: List<String>,
    messageText: String,
    onMessageTextChange: (String) -> Unit,
    onStartNode: () -> Unit,
    onSendMessage: () -> Unit,
    scope: CoroutineScope
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp),
        verticalArrangement = Arrangement.Top,
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        // Информация о ноде
        Column(
            modifier = Modifier.fillMaxWidth(),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Text(
                text = nodeId,
                style = MaterialTheme.typography.headlineSmall
            )
            if (nodeInfo.isNotEmpty()) {
                Text(
                    text = nodeInfo,
                    style = MaterialTheme.typography.bodySmall
                )
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        // Список сообщений
        Text(
            text = "Сообщения (${messages.size})",
            style = MaterialTheme.typography.titleMedium
        )

        LazyColumn(
            modifier = Modifier
                .weight(1f)
                .fillMaxWidth(),
            verticalArrangement = Arrangement.Top
        ) {
            items(messages.reversed()) { message -> // показываем новые сверху
                Card(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(4.dp),
                    elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
                ) {
                    Text(
                        text = message,
                        modifier = Modifier.padding(8.dp),
                        style = MaterialTheme.typography.bodyMedium
                    )
                }
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Поле ввода сообщения
            OutlinedTextField(
                value = messageText,
                onValueChange = onMessageTextChange,
                modifier = Modifier.weight(1f),
                placeholder = { Text("Введите сообщение...") },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Text),
                label = { Text("Сообщение") }
            )

            // Кнопка отправки
            Button(
                onClick = onSendMessage,
                enabled = messageText.isNotBlank() && nodeId.startsWith("Node ID:")
            ) {
                Text("Отправить")
            }
        }

        // Кнопки управления
        Column(
            modifier = Modifier.fillMaxWidth(),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Button(
                onClick = onStartNode,
                modifier = Modifier.fillMaxWidth(),
                enabled = !nodeId.startsWith("Node ID:")
            ) {
                Text(if (nodeId.startsWith("Node ID:")) "Нода запущена" else "Создать ноду")
            }
        }
    }
}