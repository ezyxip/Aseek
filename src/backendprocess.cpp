#include "backendprocess.h"
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <QDebug>
#include <QtEndian>
#include <QCoreApplication>

BackendProcess::BackendProcess(QObject *parent)
    : QObject(parent)
    , m_process(nullptr)
    , m_socket(nullptr)
    , m_status("stopped")
    , m_streaming(false)
    , m_reqId(0)
    , m_stageCount(0)
    , m_retryCount(0)
{
    m_retryTimer = new QTimer(this);
    m_retryTimer->setSingleShot(true);
    connect(m_retryTimer, &QTimer::timeout, this, &BackendProcess::tryConnect);

    log(QStringLiteral("BackendProcess constructed"));
}

BackendProcess::~BackendProcess()
{
    stop();
}

QString BackendProcess::status() const { return m_status; }
bool BackendProcess::isReady() const { return m_status == "ready"; }
bool BackendProcess::isBusy() const { return m_status == "busy"; }
bool BackendProcess::isStreaming() const { return m_streaming; }
QString BackendProcess::lastError() const { return m_lastError; }
QString BackendProcess::streamingText() const { return m_streamingText; }
QString BackendProcess::pipelineStage() const { return m_pipelineStage; }
QString BackendProcess::stageDetail() const { return m_stageDetail; }
int BackendProcess::stageCount() const { return m_stageCount; }

void BackendProcess::setStatus(const QString &newStatus)
{
    if (m_status != newStatus) {
        log(QStringLiteral("status: %1 -> %2").arg(m_status, newStatus));
        m_status = newStatus;
        emit statusChanged();
    }
}

void BackendProcess::setStreaming(bool value)
{
    if (m_streaming != value) {
        log(QStringLiteral("streaming: %1 -> %2").arg(m_streaming).arg(value));
        m_streaming = value;
        emit streamingChanged();
    }
}

void BackendProcess::setPipelineStage(const QString &stage, const QString &detail, int count)
{
    log(QStringLiteral("pipelineStage: stage=%1 detail=%2 count=%3").arg(stage, detail).arg(count));
    m_pipelineStage = stage;
    m_stageDetail = detail;
    m_stageCount = count;
    emit stageChanged();
}

void BackendProcess::log(const QString &message)
{
    QString ts = QDateTime::currentDateTimeUtc().toString(QStringLiteral("yyyy-MM-ddTHH:mm:ss.zzzZ"));
    QString line = QStringLiteral("[%1] %2").arg(ts, message);
    qDebug().noquote() << "[BackendProcess]" << message;
}

void BackendProcess::logEnv()
{
    log(QStringLiteral("XDG_RUNTIME_DIR=%1").arg(QString::fromUtf8(qgetenv("XDG_RUNTIME_DIR"))));
    log(QStringLiteral("AURORA_CONFIG=%1").arg(QString::fromUtf8(qgetenv("AURORA_CONFIG"))));
    log(QStringLiteral("AURORA_PROFILES=%1").arg(QString::fromUtf8(qgetenv("AURORA_PROFILES"))));
    log(QStringLiteral("AURORA_TEMPLATES=%1").arg(QString::fromUtf8(qgetenv("AURORA_TEMPLATES"))));
    log(QStringLiteral("PATH=%1").arg(QString::fromUtf8(qgetenv("PATH"))));
    log(QStringLiteral("LD_LIBRARY_PATH=%1").arg(QString::fromUtf8(qgetenv("LD_LIBRARY_PATH"))));
    log(QStringLiteral("PRIVILEGED_DIR=%1").arg(QString::fromUtf8(qgetenv("PRIVILEGED_DIR"))));
    log(QStringLiteral("APP_DIR=%1").arg(QCoreApplication::applicationDirPath()));
}

void BackendProcess::connectToServer(const QString &socketPath)
{
    if (m_socket) {
        log(QStringLiteral("connectToServer: replacing existing socket"));
        disconnectFromServer();
    }

    QString path = socketPath;
    if (path.isEmpty()) {
        path = QString::fromUtf8(qgetenv("XDG_RUNTIME_DIR")) + QStringLiteral("/aurora-rag.sock");
    }

    log(QStringLiteral("connectToServer: connecting to %1 (attempt %2)").arg(path).arg(m_retryCount));

    m_socket = new QLocalSocket(this);
    connect(m_socket, &QLocalSocket::connected, this, &BackendProcess::onConnected);
    connect(m_socket, &QLocalSocket::disconnected, this, &BackendProcess::onDisconnected);
    connect(m_socket, &QLocalSocket::readyRead, this, &BackendProcess::onReadyRead);
    connect(m_socket, static_cast<void(QLocalSocket::*)(QLocalSocket::LocalSocketError)>(&QLocalSocket::error),
            this, &BackendProcess::onError);

    m_readBuf.clear();
    m_streamingText.clear();
    m_pipelineStage.clear();
    m_stageDetail.clear();
    m_stageCount = 0;
    m_reqId = 0;

    setStatus("connecting");
    m_socket->connectToServer(path);
}

void BackendProcess::start()
{
    if (m_process) {
        qWarning() << "BackendProcess: already running";
        return;
    }

    log(QStringLiteral("start() called"));
    logEnv();

    setStatus("starting");

    QString binary = QStringLiteral("/usr/libexec/ru.pmifi.Aseek/aseek-orchestrator");
    QString cfgDir = QStringLiteral("/usr/share/ru.pmifi.Aseek/default-configs");

    log(QStringLiteral("binary=%1 exists=%2").arg(binary).arg(QFile::exists(binary)));
    log(QStringLiteral("cfgDir=%1 exists=%2").arg(cfgDir).arg(QDir(cfgDir).exists()));
    log(QStringLiteral("config=%1 exists=%2").arg(cfgDir + "/orchestrator.json").arg(QFile::exists(cfgDir + "/orchestrator.json")));
    log(QStringLiteral("profiles=%1 exists=%2").arg(cfgDir + "/profiles.json").arg(QFile::exists(cfgDir + "/profiles.json")));
    log(QStringLiteral("prompts=%1 exists=%2").arg(cfgDir + "/prompts").arg(QDir(cfgDir + "/prompts").exists()));

    m_process = new QProcess(this);
    connect(m_process, static_cast<void(QProcess::*)(int, QProcess::ExitStatus)>(&QProcess::finished),
            this, &BackendProcess::onOrchestratorFinished);
    connect(m_process, &QProcess::started,
            this, &BackendProcess::onOrchestratorStarted);
    connect(m_process, &QProcess::errorOccurred,
            this, &BackendProcess::onOrchestratorError);

    QProcessEnvironment env = QProcessEnvironment::systemEnvironment();
    env.insert(QStringLiteral("AURORA_CONFIG"), cfgDir + QStringLiteral("/orchestrator.json"));
    env.insert(QStringLiteral("AURORA_PROFILES"), cfgDir + QStringLiteral("/profiles.json"));
    env.insert(QStringLiteral("AURORA_TEMPLATES"), cfgDir + QStringLiteral("/prompts"));
    m_process->setProcessEnvironment(env);

    log(QStringLiteral("starting orchestrator..."));
    m_process->start(binary, QStringList());
}

void BackendProcess::onOrchestratorStarted()
{
    log(QStringLiteral("orchestrator started (PID=%1), will retry connection").arg(m_process->processId()));
    m_retryCount = 0;
    retryConnect();
}

void BackendProcess::onOrchestratorError(QProcess::ProcessError error)
{
    Q_UNUSED(error)
    QString errStr = m_process->errorString();
    log(QStringLiteral("orchestrator process error: %1 (code=%2)").arg(errStr).arg(error));
    m_lastError = QStringLiteral("Orchestrator error: ") + errStr;
    emit errorOccurred(m_lastError);
    stop();
}

void BackendProcess::retryConnect()
{
    int delay = 500;
    for (int i = 0; i < m_retryCount && i < 5; i++)
        delay *= 2;
    if (delay > 16000)
        delay = 16000;

    log(QStringLiteral("retryConnect: attempt %1, delay=%2ms").arg(m_retryCount + 1).arg(delay));
    m_retryTimer->setInterval(delay);
    m_retryTimer->start();
    m_retryCount++;
}

void BackendProcess::tryConnect()
{
    log(QStringLiteral("tryConnect: attempt %1").arg(m_retryCount));

    if (!m_process || m_process->state() != QProcess::Running) {
        log(QStringLiteral("tryConnect: orchestrator not running (state=%1)").arg(m_process ? QString::number(m_process->state()) : QStringLiteral("null")));
        m_lastError = QStringLiteral("Orchestrator is not running");
        emit errorOccurred(m_lastError);
        setStatus("error");
        return;
    }

    if (m_retryCount > 10) {
        log(QStringLiteral("tryConnect: max retries reached"));
        m_lastError = QStringLiteral("Could not connect to orchestrator after ") + QString::number(m_retryCount) + QStringLiteral(" attempts");
        emit errorOccurred(m_lastError);
        setStatus("error");
        return;
    }

    connectToServer();
}

void BackendProcess::stop()
{
    log(QStringLiteral("stop() called"));

    m_retryTimer->stop();

    if (m_socket) {
        log(QStringLiteral("stop: disconnecting socket"));
        m_socket->disconnect();
        m_socket->disconnectFromServer();
        m_socket->deleteLater();
        m_socket = nullptr;
    }

    if (m_process) {
        log(QStringLiteral("stop: terminating orchestrator"));
        if (m_process->state() == QProcess::Running) {
            m_process->terminate();
            if (!m_process->waitForFinished(5000)) {
                log(QStringLiteral("stop: kill after timeout"));
                m_process->kill();
                m_process->waitForFinished(3000);
            }
        }
        m_process->deleteLater();
        m_process = nullptr;
    }

    m_readBuf.clear();
    m_streamingText.clear();
    setStreaming(false);
    setPipelineStage(QString(), QString(), 0);
    setStatus("stopped");
    emit backendStopped();
}

void BackendProcess::disconnectFromServer()
{
    if (!m_socket) return;

    log(QStringLiteral("disconnectFromServer"));
    m_socket->disconnect();
    m_socket->disconnectFromServer();
    m_socket->deleteLater();
    m_socket = nullptr;
    m_readBuf.clear();
    m_streamingText.clear();
    setStreaming(false);
    setPipelineStage(QString(), QString(), 0);
    setStatus("stopped");
    emit backendStopped();
}

void BackendProcess::onConnected()
{
    log(QStringLiteral("connected to socket"));
    setStatus("ready");
    emit backendReady();
}

void BackendProcess::onDisconnected()
{
    log(QStringLiteral("socket disconnected"));
    if (!m_socket) return;
    m_socket->deleteLater();
    m_socket = nullptr;
    m_readBuf.clear();
    m_streamingText.clear();
    setStreaming(false);
    setPipelineStage(QString(), QString(), 0);
    setStatus("stopped");
    emit backendStopped();
}

void BackendProcess::onError(QLocalSocket::LocalSocketError error)
{
    log(QStringLiteral("socket error: code=%1, message=%2").arg(error).arg(m_socket ? m_socket->errorString() : QStringLiteral("no socket")));
    if (m_socket) {
        m_lastError = m_socket->errorString();
        emit errorOccurred(m_lastError);
        m_socket->deleteLater();
        m_socket = nullptr;

        log(QStringLiteral("onError: process running=%1, retryCount=%2").arg(m_process && m_process->state() == QProcess::Running).arg(m_retryCount));
        if (m_process && m_process->state() == QProcess::Running && m_retryCount <= 10) {
            retryConnect();
            return;
        }
    }
    setStatus("error");
}

void BackendProcess::sendQuery(const QString &text)
{
    if (m_status != "ready") {
        m_lastError = QStringLiteral("Backend not ready (") + m_status + QStringLiteral(")");
        log(QStringLiteral("sendQuery: ignored, status=%1").arg(m_status));
        emit errorOccurred(m_lastError);
        return;
    }

    log(QStringLiteral("sendQuery: reqId=%1, text=\"%2\"").arg(m_reqId + 1).arg(text.left(80)));
    m_currentQuery = text;
    m_streamingText.clear();
    emit streamingTextChanged();
    m_pipelineStage.clear();
    m_stageDetail.clear();
    m_stageCount = 0;
    emit stageChanged();
    setStreaming(false);

    m_reqId++;
    setStatus("busy");
    QByteArray payload = text.toUtf8();
    sendMessage(TypeQuery, payload);
}

void BackendProcess::cancelQuery()
{
    log(QStringLiteral("cancelQuery, status=%1").arg(m_status));
    if (m_status != "busy") return;

    m_streamingText.clear();
    emit streamingTextChanged();
    setStreaming(false);
    sendMessage(TypeCancel, QByteArray());
}

void BackendProcess::requestProfiles()
{
    log(QStringLiteral("requestProfiles"));
    sendMessage(TypeProfileList, QByteArray());
}

void BackendProcess::switchProfile(const QString &name)
{
    log(QStringLiteral("switchProfile: %1").arg(name));
    QJsonObject obj;
    obj[QStringLiteral("name")] = name;
    QByteArray payload = QJsonDocument(obj).toJson(QJsonDocument::Compact);
    sendMessage(TypeProfileSwitch, payload);
}

void BackendProcess::ping()
{
    log(QStringLiteral("ping"));
    sendMessage(TypePing, QByteArray());
}

bool BackendProcess::readHeader()
{
    if (m_readBuf.size() < HEADER_SIZE) return false;

    const uchar *data = reinterpret_cast<const uchar *>(m_readBuf.constData());

    TlvHeader h;
    h.magic = qFromBigEndian<quint16>(data);
    h.version = qFromBigEndian<quint16>(data + 2);
    h.type = qFromBigEndian<quint32>(data + 4);
    h.length = qFromBigEndian<quint32>(data + 8);
    h.requestId = qFromBigEndian<quint32>(data + 12);

    if (h.magic != MAGIC) {
        log(QStringLiteral("bad magic: 0x%1").arg(h.magic, 4, 16, QLatin1Char('0')));
        return false;
    }
    if (h.version != VERSION) {
        log(QStringLiteral("bad version: %1").arg(h.version));
        return false;
    }

    if (m_readBuf.size() < HEADER_SIZE + static_cast<int>(h.length)) return false;

    QByteArray payload = m_readBuf.mid(HEADER_SIZE, h.length);
    m_readBuf = m_readBuf.mid(HEADER_SIZE + h.length);

    log(QStringLiteral("read: type=%1 len=%2 reqId=%3").arg(h.type).arg(h.length).arg(h.requestId));
    handleMessage(h.type, payload);
    return true;
}

void BackendProcess::handleMessage(uint32_t type, const QByteArray &payload)
{
    switch (type) {
    case TypeToken: {
        if (!m_streaming) {
            log(QStringLiteral("recv: TypeToken (start streaming)"));
            setStreaming(true);
        }
        m_streamingText += QString::fromUtf8(payload);
        emit streamingTextChanged();
        break;
    }

    case TypeError: {
        QJsonDocument doc = QJsonDocument::fromJson(payload);
        QString msg;
        if (doc.isObject()) {
            msg = doc.object().value(QStringLiteral("message")).toString();
        }
        if (msg.isEmpty()) {
            msg = QString::fromUtf8(payload);
        }
        log(QStringLiteral("recv: TypeError: %1").arg(msg));
        m_lastError = msg;
        m_streamingText.clear();
        emit streamingTextChanged();
        setStreaming(false);
        setPipelineStage("error", msg, 0);
        setStatus("ready");
        emit errorOccurred(msg);
        break;
    }

    case TypeBusy: {
        log(QStringLiteral("recv: TypeBusy"));
        m_lastError = QStringLiteral("Server is busy");
        emit errorOccurred(m_lastError);
        break;
    }

    case TypeDone: {
        log(QStringLiteral("recv: TypeDone"));
        setStreaming(false);
        setPipelineStage("done", QString(), 0);
        setStatus("ready");
        emit queryFinished(m_currentQuery);
        break;
    }

    case TypePong:
        log(QStringLiteral("recv: TypePong"));
        emit pongReceived();
        break;

    case TypeProfileList: {
        log(QStringLiteral("recv: TypeProfileList"));
        QJsonDocument doc = QJsonDocument::fromJson(payload);
        if (doc.isArray()) {
            emit profilesReceived(doc.array());
        }
        break;
    }

    case TypeSources: {
        log(QStringLiteral("recv: TypeSources"));
        QJsonDocument doc = QJsonDocument::fromJson(payload);
        if (doc.isArray()) {
            emit sourcesReceived(doc.array());
        }
        break;
    }

    case TypeStage: {
        QJsonDocument doc = QJsonDocument::fromJson(payload);
        if (doc.isObject()) {
            QJsonObject obj = doc.object();
            QString stage = obj.value(QStringLiteral("stage")).toString();
            QString detail = obj.value(QStringLiteral("detail")).toString();
            int count = obj.value(QStringLiteral("count")).toInt();
            log(QStringLiteral("recv: TypeStage stage=%1 detail=%2 count=%3").arg(stage, detail).arg(count));
            setPipelineStage(stage, detail, count);
        }
        break;
    }

    default:
        log(QStringLiteral("recv: unknown type=%1").arg(type));
        break;
    }
}

void BackendProcess::sendMessage(uint32_t type, const QByteArray &payload)
{
    if (!m_socket || !m_socket->isOpen()) {
        log(QStringLiteral("send: type=%1 FAILED - socket not connected").arg(type));
        return;
    }

    log(QStringLiteral("send: type=%1 len=%2 reqId=%3").arg(type).arg(payload.size()).arg(m_reqId));

    QByteArray buf;
    buf.resize(HEADER_SIZE + payload.size());

    uchar *data = reinterpret_cast<uchar *>(buf.data());
    qToBigEndian<quint16>(MAGIC, data);
    qToBigEndian<quint16>(VERSION, data + 2);
    qToBigEndian<quint32>(type, data + 4);
    qToBigEndian<quint32>(payload.size(), data + 8);
    qToBigEndian<quint32>(m_reqId, data + 12);

    if (!payload.isEmpty()) {
        buf.replace(HEADER_SIZE, payload.size(), payload);
    }

    m_socket->write(buf);
    m_socket->flush();
}

void BackendProcess::onReadyRead()
{
    if (!m_socket) return;

    QByteArray data = m_socket->readAll();
    log(QStringLiteral("onReadyRead: %1 bytes").arg(data.size()));
    m_readBuf += data;

    while (readHeader())
        ;
}

void BackendProcess::onOrchestratorFinished(int exitCode, QProcess::ExitStatus exitStatus)
{
    log(QStringLiteral("orchestrator finished: code=%1 exitStatus=%2").arg(exitCode).arg(exitStatus));

    if (exitStatus == QProcess::CrashExit) {
        m_lastError = QStringLiteral("Orchestrator crashed");
        emit errorOccurred(m_lastError);
    } else if (exitCode != 0) {
        m_lastError = QStringLiteral("Orchestrator exited with code ") + QString::number(exitCode);
        emit errorOccurred(m_lastError);
    }

    m_process->deleteLater();
    m_process = nullptr;

    if (m_socket) {
        m_socket->disconnect();
        m_socket->disconnectFromServer();
        m_socket->deleteLater();
        m_socket = nullptr;
    }

    m_readBuf.clear();
    m_streamingText.clear();
    setStreaming(false);
    setPipelineStage(QString(), QString(), 0);
    setStatus("stopped");
    emit backendStopped();
}
