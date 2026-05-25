#include <QtQuick>
#include <auroraapp.h>
#include "markdownconverter.h"
#include "backendprocess.h"

int main(int argc, char *argv[])
{
    QScopedPointer<QGuiApplication> application(Aurora::Application::application(argc, argv));
    application->setOrganizationName(QStringLiteral("ru.template"));
    application->setApplicationName(QStringLiteral("Aseek"));

    qmlRegisterType<MarkdownConverter>("ru.aseek.utils", 1, 0, "MDConverter");
    qmlRegisterType<BackendProcess>("ru.aseek.utils", 1, 0, "BackendProcess");

    QScopedPointer<QQuickView> view(Aurora::Application::createView());
    view->setSource(Aurora::Application::pathTo(QStringLiteral("qml/Aseek.qml")));
    view->show();

    return application->exec();
}
