import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    objectName: "aboutPage"
    allowedOrientations: Orientation.All

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: contentColumn.height + Theme.paddingLarge

        Column {
            id: contentColumn
            width: parent.width
            spacing: Theme.paddingMedium

            PageHeader {
                objectName: "aboutPageHeader"
                title: qsTr("О приложении")
            }

            Image {
                source: Qt.resolvedUrl("../icons/Aseek.svg")
                anchors.horizontalCenter: parent.horizontalCenter
                width: Theme.iconSizeLarge
                height: width
                fillMode: Image.PreserveAspectFit
            }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                horizontalAlignment: Text.AlignHCenter
                color: Theme.primaryColor
                font.pixelSize: Theme.fontSizeMedium
                font.bold: true
                text: qsTr("Aseek")
            }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                horizontalAlignment: Text.AlignHCenter
                color: Theme.secondaryColor
                font.pixelSize: Theme.fontSizeExtraSmall
                wrapMode: Text.Wrap
                text: qsTr("Семантический поиск с RAG на устройстве")
            }

            SectionHeader {
                text: qsTr("Описание")
            }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                color: Theme.primaryColor
                font.pixelSize: Theme.fontSizeSmall
                wrapMode: Text.Wrap
                text: qsTr("Aseek — приложение для поиска информации в документах с использованием RAG-пайплайна и LLM. Запросы обрабатываются на устройстве через llama.cpp, поиск выполняется по внешним search-серверам.")
                bottomPadding: Theme.paddingMedium
            }

            SectionHeader {
                text: qsTr("Лицензия")
            }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                color: Theme.primaryColor
                font.pixelSize: Theme.fontSizeSmall
                wrapMode: Text.Wrap
                text: qsTr("Распространяется под лицензией BSD 3-Clause.\n\nCopyright (c) 2025, ru.pmifi.\nAll rights reserved.")
            }
        }
    }
}