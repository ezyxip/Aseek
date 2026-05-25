#include <QtQuick>
#include <auroraapp.h>

#include "backendprocess.h"
#include "markdownconverter.h"

int main(int argc, char *argv[])
{
    QScopedPointer<QGuiApplication> application(Aurora::Application::application(argc, argv));
    application->setOrganizationName(QStringLiteral("ru.pmifi"));
    application->setApplicationName(QStringLiteral("Aseek"));

    qmlRegisterType<BackendProcess>("ru.aseek.utils", 1, 0, "BackendProcess");
    qmlRegisterType<MarkdownConverter>("ru.aseek.utils", 1, 0, "MDConverter");

    QScopedPointer<QQuickView> view(Aurora::Application::createView());
    view->setSource(Aurora::Application::pathTo(QStringLiteral("qml/Aseek.qml")));
    view->show();

    return application->exec();
}