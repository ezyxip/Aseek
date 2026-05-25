import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: editProfilePage
    property int profileIndex
    property bool isNewProfile: false

    readonly property real dP: Screen.width / 402

    property string tempName: isNewProfile ? "" : profilesModel.get(profileIndex).name
    property string tempContext: isNewProfile ? "" : profilesModel.get(profileIndex).context

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
                width: parent.width; height: (18 + 10) * dP
                Label {
                    text: "Название профиля"
                    font.pixelSize: 18 * dP
                    color: "black"
                    anchors.left: parent.left; anchors.leftMargin: 16 * dP
                }
            }

            Item {
                width: parent.width
                height: (33 + 10) * dP
                Rectangle {
                    width: 370 * dP
                    height: 33 * dP
                    anchors.horizontalCenter: parent.horizontalCenter
                    color: "white"
                    radius: 6 * dP

                    border.color: "#2e67f2"
                    border.width: 2 * dP

                    TextInput {
                        anchors.fill: parent
                        anchors.leftMargin: 15 * dP
                        verticalAlignment: TextInput.AlignVCenter
                        text: tempName
                        font.pixelSize: 15 * dP
                        color: "black"
                        onTextChanged: tempName = text
                    }
                }
            }

            Item {
                width: parent.width
                height: isNewProfile ? 0 : (24 + 10) * dP
                visible: !isNewProfile

                Rectangle {
                    width: 370 * dP; height: 24 * dP
                    anchors.horizontalCenter: parent.horizontalCenter
                    color: "transparent"

                    Label {
                        text: "MCP-сервера"
                        font.pixelSize: 18 * dP; color: "black"
                        anchors.left: parent.left; anchors.verticalCenter: parent.verticalCenter
                    }

                    Image {
                        source: Qt.resolvedUrl("../../images/plus_icon.png")
                        width: 24 * dP; height: 24 * dP
                        anchors.right: parent.right; anchors.verticalCenter: parent.verticalCenter
                        MouseArea {
                            anchors.fill: parent
                            onClicked: pageStack.push(Qt.resolvedUrl("EditMcpPage.qml"), {
                                "profileIndex": profileIndex, "isNewServer": true
                            })
                        }
                    }
                }
            }

            Repeater {
                model: isNewProfile ? [] : profilesModel.get(profileIndex).mcpServers
                delegate: Item {
                    width: parent.width
                    height: (36 + 10) * dP
                    visible: !isNewProfile

                    Rectangle {
                        width: 370 * dP
                        height: 36 * dP
                        anchors.horizontalCenter: parent.horizontalCenter
                        color: "white"
                        radius: 4 * dP

                        Label {
                            text: model.name
                            font.pixelSize: 14 * dP
                            color: "black"
                            anchors.left: parent.left
                            anchors.leftMargin: 10 * dP
                            anchors.verticalCenter: parent.verticalCenter
                        }

                        Image {
                            source: Qt.resolvedUrl("../../images/edit_pencil.png")
                            width: 16 * dP
                            height: 16 * dP
                            anchors.right: parent.right
                            anchors.rightMargin: 10 * dP
                            anchors.verticalCenter: parent.verticalCenter

                            MouseArea {
                                anchors.fill: parent
                                onClicked: {
                                    pageStack.push(Qt.resolvedUrl("EditMcpPage.qml"), {
                                        "profileIndex": profileIndex,
                                        "serverIndex": index,
                                        "isNewServer": false
                                    })
                                }
                            }
                        }
                    }
                }
            }

            Item {
                width: parent.width; height: (18 + 14) * dP
                Label {
                    text: "Контекст"
                    font.pixelSize: 18 * dP; color: "black"
                    anchors.left: parent.left; anchors.leftMargin: 16 * dP
                }
            }

            Item {
                width: parent.width
                height: (71 + 50) * dP
                Rectangle {
                    width: 362 * dP
                    height: 71 * dP
                    anchors.horizontalCenter: parent.horizontalCenter
                    color: "white"
                    radius: 8 * dP

                    border.color: "#2e67f2"
                    border.width: 2 * dP

                    TextEdit {
                        anchors.fill: parent
                        anchors.margins: 10 * dP
                        text: tempContext
                        font.pixelSize: 14 * dP
                        color: "black"
                        wrapMode: TextEdit.Wrap
                        onTextChanged: tempContext = text
                    }
                }
            }

            Rectangle {
                width: 370 * dP; height: 35 * dP
                anchors.horizontalCenter: parent.horizontalCenter
                color: "#2e67f2"; radius: 6 * dP
                Label {
                    text: isNewProfile ? "Добавить профиль" : "Сохранить изменения"
                    color: "white"; anchors.centerIn: parent; font.pixelSize: 15 * dP
                }
                MouseArea {
                    anchors.fill: parent
                    onClicked: {
                        if (isNewProfile) {
                            profilesModel.append({ "name": tempName, "context": tempContext, "mcpServers": [] })
                        } else {
                            profilesModel.setProperty(profileIndex, "name", tempName)
                            profilesModel.setProperty(profileIndex, "context", tempContext)
                        }
                        pageStack.pop()
                    }
                }
            }

            Item { width: 1; height: 20 * dP }

            Rectangle {
                width: 370 * dP; height: 35 * dP
                anchors.horizontalCenter: parent.horizontalCenter
                color: "#ff4d4d"; radius: 6 * dP
                visible: !isNewProfile
                Label {
                    text: "Удалить профиль"
                    color: "white"; anchors.centerIn: parent; font.pixelSize: 15 * dP

                }
                MouseArea {
                    anchors.fill: parent
                    onClicked: {
                        pageStack.pop()
                        profilesModel.remove(profileIndex)
                        if (profilesModel.count === 0) {
                            profilesModel.append({ name: "Profile1", context: "", mcpServers: [] })
                        }
                        currentProfileIndex = 0
                    }
                }
            }
        }
    }
}