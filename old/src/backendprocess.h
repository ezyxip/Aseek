#ifndef BACKENDPROCESS_H
#define BACKENDPROCESS_H

#include <QObject>
#include <QProcess>
#include <QJsonObject>

class BackendProcess : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QString status READ status NOTIFY statusChanged)
    Q_PROPERTY(bool ready READ isReady NOTIFY statusChanged)
    Q_PROPERTY(bool busy READ isBusy NOTIFY statusChanged)
    Q_PROPERTY(QString lastError READ lastError NOTIFY errorOccurred)
    Q_PROPERTY(QString streamingText READ streamingText NOTIFY streamingTextChanged)
    Q_PROPERTY(bool streaming READ isStreaming NOTIFY streamingChanged)

public:
    explicit BackendProcess(QObject *parent = nullptr);
    ~BackendProcess() override;

    QString status() const;
    bool isReady() const;
    bool isBusy() const;
    bool isStreaming() const;
    QString lastError() const;
    QString streamingText() const;

    Q_INVOKABLE void start();
    Q_INVOKABLE void stop();

    // stream=true  → stream_start/stream_chunk/stream_end + summarizeResult в конце
    // stream=false → один ok + summarizeResult (как раньше)
    // language=""   → бэкенд решает сам (автодетект или --language)
    // topK=0        → бэкенд берёт свой дефолт (--top-k)
    Q_INVOKABLE void summarize(const QString &query,
                               bool stream = false,
                               const QString &language = QString(),
                               int topK = 0);

    Q_INVOKABLE void search(const QString &query, int topK = 0);

signals:
    void statusChanged();
    void errorOccurred(const QString &message);

    // --- Стриминг ---
    void streamingTextChanged();          // m_streamingText обновился
    void streamingChanged();              // m_streaming изменился
    void streamStarted();                 // получен stream_start
    void streamChunkReceived(const QString &text);  // каждый новый кусок
    void streamFinished();                // получен stream_end

    // --- Результаты ---
    void summarizeResult(const QString &query,
                         const QVariantList &results,
                         const QString &description);
    void searchResult(const QString &query,
                      const QVariantList &results);

    void backendReady();
    void backendStopped();

private slots:
    void onReadyRead();
    void onProcessFinished(int exitCode, QProcess::ExitStatus exitStatus);
    void onProcessError(QProcess::ProcessError error);

private:
    void sendCommand(const QJsonObject &cmd);
    void handleMessage(const QJsonObject &msg);
    void setStatus(const QString &newStatus);
    void setLastError(const QString &err);
    void setStreaming(bool value);

    // Общий хелпер: парсит results-массив из data-объекта
    QVariantList parseResults(const QJsonObject &data) const;

    QProcess *m_process;
    QString m_status;        // "stopped", "starting", "ready", "busy"
    QString m_lastError;
    QByteArray m_buffer;

    // Стриминг
    bool m_streaming;
    QString m_streamingText; // накапливается по чанкам, QML биндится сюда
};

#endif // BACKENDPROCESS_H
