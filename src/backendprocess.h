#ifndef BACKENDPROCESS_H
#define BACKENDPROCESS_H

#include <QObject>
#include <QProcess>
#include <QLocalSocket>
#include <QTimer>
#include <QJsonArray>
#include <QtGlobal>
#include <QFile>
#include <QTextStream>
#include <QDateTime>
#include <QDir>

class BackendProcess : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QString status READ status NOTIFY statusChanged)
    Q_PROPERTY(bool ready READ isReady NOTIFY statusChanged)
    Q_PROPERTY(bool busy READ isBusy NOTIFY statusChanged)
    Q_PROPERTY(QString lastError READ lastError NOTIFY errorOccurred)
    Q_PROPERTY(QString streamingText READ streamingText NOTIFY streamingTextChanged)
    Q_PROPERTY(bool streaming READ isStreaming NOTIFY streamingChanged)
    Q_PROPERTY(QString pipelineStage READ pipelineStage NOTIFY stageChanged)
    Q_PROPERTY(QString stageDetail READ stageDetail NOTIFY stageChanged)
    Q_PROPERTY(int stageCount READ stageCount NOTIFY stageChanged)

public:
    explicit BackendProcess(QObject *parent = nullptr);
    ~BackendProcess() override;

    QString status() const;
    bool isReady() const;
    bool isBusy() const;
    bool isStreaming() const;
    QString lastError() const;
    QString streamingText() const;
    QString pipelineStage() const;
    QString stageDetail() const;
    int stageCount() const;

    Q_INVOKABLE void start();
    Q_INVOKABLE void stop();
    Q_INVOKABLE void connectToServer(const QString &socketPath = QString());
    Q_INVOKABLE void disconnectFromServer();
    Q_INVOKABLE void sendQuery(const QString &text);
    Q_INVOKABLE void cancelQuery();
    Q_INVOKABLE void requestProfiles();
    Q_INVOKABLE void switchProfile(const QString &name);
    Q_INVOKABLE void ping();

signals:
    void statusChanged();
    void errorOccurred(const QString &message);
    void streamingTextChanged();
    void streamingChanged();
    void stageChanged();

    void queryFinished(const QString &query);
    void sourcesReceived(const QJsonArray &sources);
    void profilesReceived(const QJsonArray &profiles);
    void backendReady();
    void backendStopped();
    void pongReceived();

private slots:
    void onConnected();
    void onDisconnected();
    void onReadyRead();
    void onError(QLocalSocket::LocalSocketError error);
    void onOrchestratorFinished(int exitCode, QProcess::ExitStatus exitStatus);
    void onOrchestratorStarted();
    void onOrchestratorError(QProcess::ProcessError error);
    void tryConnect();
    void retryConnect();

private:
    void log(const QString &message);
    void logEnv();

    struct TlvHeader {
        quint16 magic;
        quint16 version;
        quint32 type;
        quint32 length;
        quint32 requestId;
    };

    static const quint16 MAGIC = 0x4152;
    static const quint16 VERSION = 1;
    static const int HEADER_SIZE = 16;

    enum MessageType {
        TypeQuery        = 1,
        TypeToken        = 2,
        TypeError        = 3,
        TypeBusy         = 4,
        TypeDone         = 5,
        TypeCancel       = 6,
        TypePing         = 7,
        TypePong         = 8,
        TypeProfileList  = 9,
        TypeProfileSwitch = 10,
        TypeSources      = 11,
        TypeStage        = 12,
    };

    bool readHeader();
    void handleMessage(quint32 type, const QByteArray &payload);
    void sendMessage(quint32 type, const QByteArray &payload);
    void setStatus(const QString &newStatus);
    void setStreaming(bool value);
    void setPipelineStage(const QString &stage, const QString &detail, int count);

    QProcess *m_process;
    QLocalSocket *m_socket;
    QFile *m_logFile;
    QTextStream m_logStream;
    QByteArray m_readBuf;
    QString m_status;
    QString m_lastError;
    QTimer *m_retryTimer;
    int m_retryCount;

    bool m_streaming;
    QString m_streamingText;
    quint32 m_reqId;
    QString m_currentQuery;

    QString m_pipelineStage;
    QString m_stageDetail;
    int m_stageCount;
};

#endif