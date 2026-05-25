import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: editMcpPage
    property int profileIndex
    property int serverIndex: -1
    property bool isNewServer: true

    readonly property real dP: Screen.width / 402

    property string tempName: isNewServer ? "" : profilesModel.get(profileIndex).mcpServers.get(serverIndex).name
    property string tempUrl: isNewServer ? "" : profilesModel.get(profileIndex).mcpServers.get(serverIndex).url
    property string tempSsoUrl: isNewServer ? "" : (profilesModel.get(profileIndex).mcpServers.get(serverIndex).ssoUrl || "")
    property string tempDescription: isNewServer ? "" : (profilesModel.get(profileIndex).mcpServers.get(serverIndex).description || "")

    Rectangle {
        anchors.fill: parent
        color: "#f8fafd"
        z: -1
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: col.height

        Column {
            id: col
            width: parent.width

            Item { width: parent.width; height: 20 * dP }

            Item {
                width: parent.width
                height: (36 + 20) * dP
                Rectangle {
                    width: 392 * dP; height: 36 * dP
                    anchors.horizontalCenter: parent.horizontalCenter
                    color: "transparent"
                    Image {
                        source: Qt.resolvedUrl("../../images/back_arrow.png")
                        width: 36 * dP; height: 36 * dP
                        MouseArea { anchors.fill: parent; onClicked: pageStack.pop() }
                    }
                }
            }

            Item {
                width: parent.width; height: (18 + 14) * dP
                Label {
                    text: "Название"; font.pixelSize: 18 * dP; color: "black"
                    anchors.left: parent.left; anchors.leftMargin: 16 * dP
                }
            }
            Item {
                width: parent.width; height: (33 + 14) * dP
                Rectangle {
                    width: 362 * dP; height: 33 * dP
                    anchors.horizontalCenter: parent.horizontalCenter
                    color: "white"; radius: 6 * dP; border.color: "#2e67f2"; border.width: 2 * dP
                    TextInput {
                        anchors.fill: parent; anchors.leftMargin: 15 * dP
                        verticalAlignment: TextInput.AlignVCenter
                        text: tempName; font.pixelSize: 15 * dP; onTextChanged: tempName = text
                    }
                }
            }

            Item {
                width: parent.width; height: (18 + 14) * dP
                Label {
                    text: "URL сервера"; font.pixelSize: 18 * dP; color: "black"
                    anchors.left: parent.left; anchors.leftMargin: 16 * dP
                }
            }
            Item {
                width: parent.width; height: (33 + 14) * dP
                Rectangle {
                    width: 362 * dP; height: 33 * dP
                    anchors.horizontalCenter: parent.horizontalCenter
                    color: "white"; radius: 6 * dP; border.color: "#2e67f2"; border.width: 2 * dP
                    TextInput {
                        anchors.fill: parent; anchors.leftMargin: 15 * dP
                        verticalAlignment: TextInput.AlignVCenter; inputMethodHints: Qt.ImhUrlCharactersOnly
                        text: tempUrl; font.pixelSize: 15 * dP; onTextChanged: tempUrl = text
                    }
                }
            }

            Item {
                width: parent.width; height: (18 + 14) * dP
                Label {
                    text: "SSO URL"; font.pixelSize: 18 * dP; color: "black"
                    anchors.left: parent.left; anchors.leftMargin: 16 * dP
                }
            }
            Item {
                width: parent.width; height: (33 + 14) * dP
                Rectangle {
                    width: 362 * dP; height: 33 * dP
                    anchors.horizontalCenter: parent.horizontalCenter
                    color: "white"; radius: 6 * dP; border.color: "#2e67f2"; border.width: 2 * dP
                    TextInput {
                        anchors.fill: parent; anchors.leftMargin: 15 * dP
                        verticalAlignment: TextInput.AlignVCenter; inputMethodHints: Qt.ImhUrlCharactersOnly
                        text: tempSsoUrl; font.pixelSize: 15 * dP; onTextChanged: tempSsoUrl = text
                    }
                }
            }

            Item {
                width: parent.width; height: (18 + 14) * dP
                Label {
                    text: "Описание сервера"; font.pixelSize: 18 * dP; color: "black"
                    anchors.left: parent.left; anchors.leftMargin: 16 * dP
                }
            }
            Item {
                width: parent.width; height: (48 + 50) * dP
                Rectangle {
                    width: 362 * dP; height: 48 * dP
                    anchors.horizontalCenter: parent.horizontalCenter
                    color: "white"; radius: 8 * dP; border.color: "#2e67f2"; border.width: 2 * dP
                    TextEdit {
                        anchors.fill: parent; anchors.margins: 8 * dP
                        wrapMode: TextEdit.Wrap; text: tempDescription
                        font.pixelSize: 14 * dP; onTextChanged: tempDescription = text
                    }
                }
            }

            Rectangle {
                width: 370 * dP; height: 35 * dP
                anchors.horizontalCenter: parent.horizontalCenter
                color: "#2e67f2"; radius: 6 * dP
                Label {
                    text: "Авторизоваться"; color: "white"; anchors.centerIn: parent; font.pixelSize: 15 * dP
                }
                MouseArea {
                    anchors.fill: parent
                    onClicked: {
                        var mcpList = profilesModel.get(profileIndex).mcpServers
                        var data = { "name": tempName, "url": tempUrl, "ssoUrl": tempSsoUrl, "description": tempDescription }
                        if (isNewServer) mcpList.append(data);
                        else {
                            mcpList.setProperty(serverIndex, "name", tempName)
                            mcpList.setProperty(serverIndex, "url", tempUrl)
                            mcpList.setProperty(serverIndex, "ssoUrl", tempSsoUrl)
                            mcpList.setProperty(serverIndex, "description", tempDescription)
                        }
                        pageStack.pop()
                    }
                }
            }

            Item { width: 1; height: 10 * dP }

            Rectangle {
                width: 370 * dP; height: 35 * dP
                anchors.horizontalCenter: parent.horizontalCenter
                color: "#ff4d4d"; radius: 6 * dP; visible: !isNewServer
                Label {
                    text: "Удалить сервер"; color: "white"; anchors.centerIn: parent; font.pixelSize: 15 * dP
                }
                MouseArea {
                    anchors.fill: parent
                    onClicked: {
                        profilesModel.get(profileIndex).mcpServers.remove(serverIndex)
                        pageStack.pop()
                    }
                }
            }
        }
    }
}
