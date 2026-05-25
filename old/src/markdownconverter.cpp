#include "markdownconverter.h"
#include <QVariantMap>

extern "C" {
#include "md4c-html.h"
}

void process_output(const MD_CHAR* text, MD_SIZE size, void* userdata) {
    static_cast<QString*>(userdata)->append(QString::fromUtf8(text, size));
}

MarkdownConverter::MarkdownConverter(QObject *parent) : QObject(parent) {}

QString MarkdownConverter::toHtml(const QString &markdown) {
    if (markdown.isEmpty()) return QString();
    QString htmlResult;

    htmlResult = "<style>"
                 "h1 { font-size: 1.2em; }"
                 "h2 { font-size: 1.1em; }"
                 "h3 { font-size: 1.0em; }"
                 "p { margin: 0; }"
                 "</style>";

    QByteArray ba = markdown.toUtf8();

    md_html(ba.data(), ba.size(), process_output, &htmlResult, MD_DIALECT_GITHUB, 0);
    return htmlResult;
}

QVariantList MarkdownConverter::parseToParts(const QString &markdown) {
    QVariantList parts;
    if (markdown.isEmpty()) return parts;

    QStringList rawParts = markdown.split("```");

    for (int i = 0; i < rawParts.size(); ++i) {
        QString content = rawParts[i];
        if (content.isEmpty() && i % 2 == 0) continue;

        QVariantMap item;
        if (i % 2 == 1) {
            item["isCode"] = true;
            QString codeContent = content.trimmed();

            if (codeContent.contains('\n')) {
                QString firstLine = codeContent.section('\n', 0, 0).trimmed().toLower();
                QStringList langs = {"cpp", "c", "python", "js", "qml", "bash", "sql", "java"};
                if (langs.contains(firstLine)) {
                    codeContent = codeContent.section('\n', 1); // Отрезаем первую строку
                }
            }
            item["content"] = codeContent.trimmed();
        } else {
            item["isCode"] = false;
            item["content"] = this->toHtml(content);
        }
        parts.append(item);
    }
    return parts;
}
