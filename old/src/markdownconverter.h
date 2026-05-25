#ifndef MARKDOWNCONVERTER_H
#define MARKDOWNCONVERTER_H

#include <QObject>
#include <QVariantList>
#include <QStringList>

class MarkdownConverter : public QObject
{
    Q_OBJECT
public:
    explicit MarkdownConverter(QObject *parent = nullptr);

    Q_INVOKABLE QString toHtml(const QString &markdown);

    Q_INVOKABLE QVariantList parseToParts(const QString &markdown);
};

#endif // MARKDOWNCONVERTER_H
