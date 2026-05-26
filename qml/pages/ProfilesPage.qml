import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: profilesPage
    allowedOrientations: Orientation.Portrait

    readonly property real dP: Screen.width / 402

    onStatusChanged: {
        if (status === PageStatus.Active) {
            backend.requestProfiles()
        }
    }

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
                    width: 392 * dP
                    height: 36 * dP
                    anchors.horizontalCenter: parent.horizontalCenter
                    color: "transparent"

                    Image {
                        id: backBtn
                        source: Qt.resolvedUrl("../../images/back_arrow.png")
                        width: 36 * dP
                        height: 36 * dP
                        anchors.left: parent.left
                        MouseArea {
                            anchors.fill: parent
                            onClicked: pageStack.pop()
                        }
                    }
                }
            }

            Item {
                width: parent.width
                height: (24 + 10) * dP

                Rectangle {
                    width: 370 * dP
                    height: 24 * dP
                    anchors.horizontalCenter: parent.horizontalCenter
                    color: "transparent"

                    Label {
                        text: "Профили"
                        color: "black"
                        font.pixelSize: 18 * dP
                        anchors.left: parent.left
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    Image {
                        id: addBtn
                        source: Qt.resolvedUrl("../../images/plus_icon.png")
                        width: 24 * dP
                        height: 24 * dP
                        anchors.right: parent.right
                        anchors.verticalCenter: parent.verticalCenter
                        MouseArea {
                            anchors.fill: parent
                            onClicked: {
                                pageStack.push(Qt.resolvedUrl("EditProfilePage.qml"), {
                                    "profileIndex": -1,
                                    "isNewProfile": true
                                })
                            }
                        }
                    }
                }
            }

            Repeater {
                model: profilesModel
                delegate: Item {
                    width: profilesPage.width
                    height: (36 + 10) * dP

                    Rectangle {
                        width: 370 * dP
                        height: 36 * dP
                        anchors.horizontalCenter: parent.horizontalCenter
                        radius: 6 * dP
                        color: "white"

                        border.color: currentProfileIndex === index ? "#2e67f2" : "#eef2f7"
                        border.width: currentProfileIndex === index ? 2 : 1

                        MouseArea {
                            anchors.fill: parent
                            onClicked: currentProfileIndex = index
                        }

                        Label {
                            text: model.name
                            color: "black"
                            font.pixelSize: 14 * dP
                            anchors {
                                left: parent.left
                                leftMargin: 10 * dP
                                verticalCenter: parent.verticalCenter
                            }
                            width: parent.width - 46 * dP
                            elide: Text.ElideRight
                        }

                        Image {
                            id: pencilIcon
                            source: Qt.resolvedUrl("../../images/edit_pencil.png")
                            width: 16 * dP
                            height: 16 * dP
                            anchors {
                                right: parent.right
                                rightMargin: 10 * dP
                                verticalCenter: parent.verticalCenter
                            }
                            MouseArea {
                                anchors.fill: parent
                                onClicked: {
                                    pageStack.push(Qt.resolvedUrl("EditProfilePage.qml"), {
                                        "profileIndex": index,
                                        "isNewProfile": false
                                    })
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}