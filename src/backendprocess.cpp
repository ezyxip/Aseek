#include "backendprocess.h"
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <QDebug>
#include <QtEndian>

BackendProcess::BackendProcess(QObject *parent)
    : QObject(parent)
    , m_socket(nullptr)
    , m_status("disconnected")
    , m_streaming(false)
    , m_reqId(0)
    , m_stageCount(0)
{
}

BackendProcess::~BackendProcess()
{
    disconnectFromServer();
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
        m_status = newStatus;
        emit statusChanged();
    }
}

void BackendProcess::setStreaming(bool value)
{
    if (m_streaming != value) {
        m_streaming = value;
        emit streamingChanged();
    }
}

void BackendProcess::setPipelineStage(const QString &stage, const QString &detail, int count)
{
    m_pipelineStage = stage;
    m_stageDetail = detail;
    m_stageCount = count;
    emit stageChanged();
}

void BackendProcess::connectToServer(const QString &socketPath)
{
    if (m_socket) {
        disconnectFromServer();
    }

    QString path = socketPath;
    if (path.isEmpty()) {
        path = QString::fromUtf8(qgetenv("XDG_RUNTIME_DIR")) + QStringLiteral("/aurora-rag.sock");
    }

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

void BackendProcess::disconnectFromServer()
{
    if (!m_socket) return;

    m_socket->disconnectFromServer();
    m_socket->deleteLater();
    m_socket = nullptr;
    m_readBuf.clear();
    m_streamingText.clear();
    setStreaming(false);
    setPipelineStage(QString(), QString(), 0);
    setStatus("disconnected");
    emit backendStopped();
}

void BackendProcess::onConnected()
{
    setStatus("ready");
    emit backendReady();
}

void BackendProcess::onDisconnected()
{
    if (!m_socket) return;
    m_socket->deleteLater();
    m_socket = nullptr;
    m_readBuf.clear();
    m_streamingText.clear();
    setStreaming(false);
    setPipelineStage(QString(), QString(), 0);
    setStatus("disconnected");
    emit backendStopped();
}

void BackendProcess::onError(QLocalSocket::LocalSocketError error)
{
    Q_UNUSED(error)
    if (m_socket) {
        m_lastError = m_socket->errorString();
        emit errorOccurred(m_lastError);
    }
    setStatus("error");
}

void BackendProcess::sendQuery(const QString &text)
{
    if (m_status != "ready" && m_status != "idle") {
        m_lastError = QStringLiteral("Backend not ready (") + m_status + QStringLiteral(")");
        emit errorOccurred(m_lastError);
        return;
    }

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
    if (m_status != "busy") return;

    QJsonObject obj;
    obj[QStringLiteral("status")] = QStringLiteral("cancelling");
    m_streamingText.clear();
    emit streamingTextChanged();
    setStreaming(false);
    sendMessage(TypeCancel, QByteArray());
}

void BackendProcess::requestProfiles()
{
    sendMessage(TypeProfileList, QByteArray());
}

void BackendProcess::switchProfile(const QString &name)
{
    QJsonObject obj;
    obj[QStringLiteral("name")] = name;
    QByteArray payload = QJsonDocument(obj).toJson(QJsonDocument::Compact);
    sendMessage(TypeProfileSwitch, payload);
}

void BackendProcess::ping()
{
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
        qWarning() << "BackendProcess: bad magic" << hex << h.magic;
        return false;
    }
    if (h.version != VERSION) {
        qWarning() << "BackendProcess: bad version" << h.version;
        return false;
    }

    if (m_readBuf.size() < HEADER_SIZE + static_cast<int>(h.length)) return false;

    QByteArray payload = m_readBuf.mid(HEADER_SIZE, h.length);
    m_readBuf = m_readBuf.mid(HEADER_SIZE + h.length);

    handleMessage(h.type, payload);
    return true;
}

void BackendProcess::handleMessage(uint32_t type, const QByteArray &payload)
{
    switch (type) {
    case TypeToken: {
        if (!m_streaming) {
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
        m_lastError = QStringLiteral("Server is busy");
        emit errorOccurred(m_lastError);
        break;
    }

    case TypeDone: {
        setStreaming(false);
        setPipelineStage("done", QString(), 0);
        setStatus("ready");
        emit queryFinished(m_currentQuery);
        break;
    }

    case TypePong:
        emit pongReceived();
        break;

    case TypeProfileList: {
        QJsonDocument doc = QJsonDocument::fromJson(payload);
        if (doc.isArray()) {
            emit profilesReceived(doc.array());
        }
        break;
    }

    case TypeSources: {
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
            setPipelineStage(stage, detail, count);
        }
        break;
    }

    default:
        qWarning() << "BackendProcess: unknown message type" << type;
        break;
    }
}

void BackendProcess::sendMessage(uint32_t type, const QByteArray &payload)
{
    if (!m_socket || !m_socket->isOpen()) {
        qWarning() << "BackendProcess: socket not connected";
        return;
    }

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

    m_readBuf += m_socket->readAll();

    while (readHeader())
        ;
}
