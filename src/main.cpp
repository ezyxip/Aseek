#include <QtQuick>
#include <auroraapp.h>

int main(int argc, char *argv[])
{
    QScopedPointer<QGuiApplication> application(Aurora::Application::application(argc, argv));
    application->setOrganizationName(QStringLiteral("ru.pmifi"));
    application->setApplicationName(QStringLiteral("Aseek"));

    QScopedPointer<QQuickView> view(Aurora::Application::createView());
    view->setSource(Aurora::Application::pathTo(QStringLiteral("qml/Aseek.qml")));
    view->show();

    return application->exec();
}
