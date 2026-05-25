#include "backendprocess.h"
#include <QJsonDocument>
#include <QJsonArray>
#include <QDebug>

// ────────────────────────────────────────────────────────────
// Конструктор / деструктор
// ────────────────────────────────────────────────────────────

BackendProcess::BackendProcess(QObject *parent)
    : QObject(parent)
    , m_process(nullptr)
    , m_status("stopped")
    , m_streaming(false)
{
}

BackendProcess::~BackendProcess()
{
    stop();
}

// ────────────────────────────────────────────────────────────
// Свойства
// ────────────────────────────────────────────────────────────

QString BackendProcess::status() const { return m_status; }
bool BackendProcess::isReady() const { return m_status == "ready"; }
bool BackendProcess::isBusy() const { return m_status == "busy"; }
bool BackendProcess::isStreaming() const { return m_streaming; }
QString BackendProcess::lastError() const { return m_lastError; }
QString BackendProcess::streamingText() const { return m_streamingText; }

void BackendProcess::setStatus(const QString &newStatus)
{
    if (m_status != newStatus) {
        m_status = newStatus;
        emit statusChanged();
    }
}

void BackendProcess::setLastError(const QString &err)
{
    m_lastError = err;
    emit errorOccurred(err);
}

void BackendProcess::setStreaming(bool value)
{
    if (m_streaming != value) {
        m_streaming = value;
        emit streamingChanged();
    }
}

// ────────────────────────────────────────────────────────────
// Жизненный цикл процесса
// ────────────────────────────────────────────────────────────

void BackendProcess::start()
{
    if (m_process) {
        qWarning() << "BackendProcess: already running";
        return;
    }

    m_process = new QProcess(this);
    m_buffer.clear();
    m_streamingText.clear();

    connect(m_process, &QProcess::readyReadStandardOutput,
            this, &BackendProcess::onReadyRead);
    connect(m_process, static_cast<void(QProcess::*)(int, QProcess::ExitStatus)>(&QProcess::finished),
            this, &BackendProcess::onProcessFinished);
    connect(m_process, &QProcess::errorOccurred,
            this, &BackendProcess::onProcessError);

    m_process->setProcessChannelMode(QProcess::ForwardedErrorChannel);

    QString binary = QStringLiteral("/usr/libexec/ru.omgtu.aseek/bin/aseek-backend");
    QString modelPath = QStringLiteral("/usr/share/ru.omgtu.aseek/models/"
                                       "Qwen_Qwen2.5-1.5B-Instruct-GGUF_qwen2.5-1.5b-instruct-q4_k_m.gguf");

    QStringList args;
    args << "--embedding-url" << "http://130.49.181.2:8000/v1/embed"
         << "--qdrant-url"    << "http://130.49.181.2:6333"
         << "--collection"    << "test"
         << "--model"         << modelPath
         << "--threads"       << "6"
         << "--llama-bin"     << "/usr/libexec/ru.omgtu.aseek/bin/llama-cli"
         << "--top-k"         << "3"
         << "--context-size"  << "3072";

    qDebug() << "BackendProcess: starting" << binary << args;

    setStatus("starting");
    m_process->start(binary, args);

    if (!m_process->waitForStarted(5000)) {
        setLastError("Failed to start backend process: " + m_process->errorString());
        setStatus("stopped");
        m_process->deleteLater();
        m_process = nullptr;
    }
}

void BackendProcess::stop()
{
    if (!m_process)
        return;

    if (m_process->state() == QProcess::Running) {
        QJsonObject cmd;
        cmd["command"] = "exit";
        sendCommand(cmd);

        if (!m_process->waitForFinished(5000)) {
            qWarning() << "BackendProcess: force killing";
            m_process->kill();
            m_process->waitForFinished(3000);
        }
    }

    m_process->deleteLater();
    m_process = nullptr;
    m_buffer.clear();
    m_streamingText.clear();
    setStreaming(false);
    setStatus("stopped");
    emit backendStopped();
}

// ────────────────────────────────────────────────────────────
// Команды
// ────────────────────────────────────────────────────────────

void BackendProcess::summarize(const QString &query, bool stream,
                               const QString &language, int topK)
{
    if (!isReady()) {
        setLastError("Backend is not ready (status: " + m_status + ")");
        return;
    }

    QJsonObject cmd;
    cmd["command"] = "summarize";
    cmd["query"]   = query;

    if (stream)
        cmd["stream"] = true;
    if (!language.isEmpty())
        cmd["language"] = language;
    if (topK > 0)
        cmd["top_k"] = topK;

    // Очищаем текст стриминга перед новым запросом
    m_streamingText.clear();
    emit streamingTextChanged();

    setStatus("busy");
    sendCommand(cmd);
}

void BackendProcess::search(const QString &query, int topK)
{
    if (!isReady()) {
        setLastError("Backend is not ready (status: " + m_status + ")");
        return;
    }

    QJsonObject cmd;
    cmd["command"] = "search";
    cmd["query"]   = query;

    if (topK > 0)
        cmd["top_k"] = topK;

    setStatus("busy");
    sendCommand(cmd);
}

void BackendProcess::sendCommand(const QJsonObject &cmd)
{
    if (!m_process || m_process->state() != QProcess::Running) {
        setLastError("Backend process is not running");
        return;
    }

    QByteArray data = QJsonDocument(cmd).toJson(QJsonDocument::Compact) + '\n';
    qDebug() << "BackendProcess: >>>" << data.trimmed();
    m_process->write(data);
}

// ────────────────────────────────────────────────────────────
// Чтение stdout (NDJSON)
// ────────────────────────────────────────────────────────────

void BackendProcess::onReadyRead()
{
    m_buffer += m_process->readAllStandardOutput();

    while (true) {
        int idx = m_buffer.indexOf('\n');
        if (idx < 0)
            break;

        QByteArray line = m_buffer.left(idx).trimmed();
        m_buffer = m_buffer.mid(idx + 1);

        if (line.isEmpty())
            continue;

        qDebug() << "BackendProcess: <<<" << line;

        QJsonParseError parseError;
        QJsonDocument doc = QJsonDocument::fromJson(line, &parseError);
        if (parseError.error != QJsonParseError::NoError) {
            qWarning() << "BackendProcess: JSON parse error:" << parseError.errorString();
            continue;
        }

        handleMessage(doc.object());
    }
}

// ────────────────────────────────────────────────────────────
// Диспетчер сообщений
// ────────────────────────────────────────────────────────────

void BackendProcess::handleMessage(const QJsonObject &msg)
{
    const QString msgStatus = msg["status"].toString();

    // ── ready ──────────────────────────────────────────────
    if (msgStatus == "ready") {
        setStatus("ready");
        emit backendReady();
        return;
    }

    // ── error ──────────────────────────────────────────────
    if (msgStatus == "error") {
        QString errorMsg = msg["message"].toString("Unknown backend error");
        setLastError(errorMsg);
        setStreaming(false);
        setStatus("ready");
        return;
    }

    // ── stream_start ──────────────────────────────────────
    if (msgStatus == "stream_start") {
        m_streamingText.clear();
        emit streamingTextChanged();
        setStreaming(true);
        emit streamStarted();
        return;
    }

    // ── stream_chunk ──────────────────────────────────────
    if (msgStatus == "stream_chunk") {
        QJsonObject data = msg["data"].toObject();
        QString text = data["text"].toString();

        if (!text.isEmpty()) {
            m_streamingText += text;
            emit streamingTextChanged();
            emit streamChunkReceived(text);
        }
        return;
    }

    // ── stream_end ────────────────────────────────────────
    if (msgStatus == "stream_end") {
        QString command = msg["command"].toString();
        QJsonObject data = msg["data"].toObject();

        setStreaming(false);
        emit streamFinished();

        if (command == "summarize") {
            QString query       = data["query"].toString();
            QString description = data["description"].toString();
            QVariantList results = parseResults(data);

            // Обновляем streamingText финальным чистым текстом
            m_streamingText = description;
            emit streamingTextChanged();

            setStatus("ready");
            emit summarizeResult(query, results, description);
        } else {
            setStatus("ready");
        }
        return;
    }

    // ── ok (не-стриминговый ответ) ────────────────────────
    if (msgStatus == "ok") {
        QString command = msg["command"].toString();

        if (command == "summarize") {
            QJsonObject data    = msg["data"].toObject();
            QString query       = data["query"].toString();
            QString description = data["description"].toString();
            QVariantList results = parseResults(data);

            // Для единообразия записываем и в streamingText
            m_streamingText = description;
            emit streamingTextChanged();

            setStatus("ready");
            emit summarizeResult(query, results, description);
        }
        else if (command == "search") {
            QJsonObject data = msg["data"].toObject();
            QString query    = data["query"].toString();
            QVariantList results = parseResults(data);

            setStatus("ready");
            emit searchResult(query, results);
        }
        else if (command == "exit") {
            // Процесс сам завершится
        }
        else {
            setStatus("ready");
        }
        return;
    }

    qWarning() << "BackendProcess: unknown status:" << msgStatus;
}

// ────────────────────────────────────────────────────────────
// Хелперы
// ────────────────────────────────────────────────────────────

QVariantList BackendProcess::parseResults(const QJsonObject &data) const
{
    QVariantList results;
    const QJsonArray arr = data["results"].toArray();

    for (const QJsonValue &val : arr) {
        QJsonObject item = val.toObject();
        QVariantMap result;
        result["id"]    = item["id"].toInt();
        result["score"] = item["score"].toDouble();

        QJsonObject payload = item["payload"].toObject();
        QVariantMap payloadMap;
        for (auto it = payload.begin(); it != payload.end(); ++it) {
            payloadMap[it.key()] = it.value().toVariant();
        }
        result["payload"] = payloadMap;
        results.append(result);
    }

    return results;
}

// ────────────────────────────────────────────────────────────
// Обработка завершения / ошибок процесса
// ────────────────────────────────────────────────────────────

void BackendProcess::onProcessFinished(int exitCode, QProcess::ExitStatus exitStatus)
{
    qDebug() << "BackendProcess: finished, code=" << exitCode
             << "exitStatus=" << exitStatus;

    if (exitStatus == QProcess::CrashExit) {
        setLastError("Backend process crashed");
    } else if (exitCode != 0) {
        setLastError("Backend exited with code " + QString::number(exitCode));
    }

    m_process->deleteLater();
    m_process = nullptr;
    m_buffer.clear();
    m_streamingText.clear();
    setStreaming(false);
    setStatus("stopped");
    emit backendStopped();
}

void BackendProcess::onProcessError(QProcess::ProcessError error)
{
    Q_UNUSED(error)
    if (m_process) {
        setLastError("Process error: " + m_process->errorString());
    }
}
