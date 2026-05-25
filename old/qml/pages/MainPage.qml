import QtQuick 2.0
import Sailfish.Silica 1.0
import ru.aseek.utils 1.0

Page {
    id: root
    objectName: "mainPage"

    property string currentProfileName: "Загрузка..."
    readonly property real dP: Screen.width / 402
    readonly property real fontScale: 0.7
    property bool isSearching: backend.busy

    // Индекс сообщения-плейсхолдера, которое обновляется при стриминге
    property int streamingItemIndex: -1

    MDConverter { id: mdConverter }

    ListModel {
        id: chatModel
    }

    // ── Бэкенд-интеграция ────────────────────────────────────────

    Connections {
        target: backend

        onBackendReady: {
            console.log("Backend is ready")
        }

        // ── Стриминг ─────────────────────────────────────────────

        onStreamStarted: {
            console.log("Stream started")
        }

        onStreamingTextChanged: {
            // Обновляем плейсхолдер по мере поступления чанков
            if (streamingItemIndex >= 0 && streamingItemIndex < chatModel.count) {
                var displayText = backend.streamingText
                if (!displayText || displayText === "")
                    displayText = "..."
                chatModel.setProperty(streamingItemIndex, "text", displayText)
            }
        }

        onStreamFinished: {
            console.log("Stream finished")
        }

        // ── Финальный результат (стриминг и обычный режим) ────────

        onSummarizeResult: {
            // query, results, description
            if (streamingItemIndex >= 0 && streamingItemIndex < chatModel.count) {
                // Стриминг: обновляем уже существующее сообщение финальным текстом
                chatModel.setProperty(streamingItemIndex, "text", description)
                streamingItemIndex = -1
            } else {
                // Обычный режим: добавляем новое сообщение
                chatModel.append({
                    "type": "bot",
                    "text": description
                })
            }
            searchInput.text = ""
            scrollTimer.start()
        }

        onSearchResult: {
            // query, results — на будущее
            searchInput.text = ""
            scrollTimer.start()
        }

        // ── Ошибки ───────────────────────────────────────────────

        onErrorOccurred: {
            // message
            if (streamingItemIndex >= 0 && streamingItemIndex < chatModel.count) {
                // Ошибка во время стриминга: заменяем плейсхолдер ошибкой
                chatModel.setProperty(streamingItemIndex, "text",
                    "**Ошибка:** " + message)
                streamingItemIndex = -1
            } else {
                chatModel.append({
                    "type": "bot",
                    "text": "**Ошибка:** " + message
                })
            }
            searchInput.text = ""
            scrollTimer.start()
        }

        onBackendStopped: {
            console.log("Backend stopped")
            streamingItemIndex = -1
        }
    }

    // ── Поиск / стриминг ─────────────────────────────────────────

    function startSearch() {
        if (searchInput.text.trim() === "" || isSearching) return
        if (!backend.ready) {
            chatModel.append({
                "type": "bot",
                "text": "**Бэкенд ещё не готов.** Подождите..."
            })
            return
        }

        var query = searchInput.text.trim()

        // Сообщение пользователя
        chatModel.append({ "type": "user", "text": query })

        // Плейсхолдер для ответа бота — будет обновляться чанками
        chatModel.append({ "type": "bot", "text": "..." })
        streamingItemIndex = chatModel.count - 1

        searchInput.text = "подождите..."

        // stream=true для посимвольного вывода
        backend.summarize(query, true)
        scrollTimer.start()
    }

    function stopSearch() {
        // Фиксируем то, что успело прийти
        var wasStreaming = (streamingItemIndex >= 0)
        var idx = streamingItemIndex
        streamingItemIndex = -1

        if (wasStreaming && idx >= 0 && idx < chatModel.count) {
            var currentText = chatModel.get(idx).text
            if (!currentText || currentText === "...") {
                chatModel.setProperty(idx, "text", "*[Прервано]*")
            } else {
                chatModel.setProperty(idx, "text",
                    currentText + "\n\n*[Прервано]*")
            }
        }

        backend.stop()
        backend.start()
        searchInput.text = ""
    }

    // Одноразовый скролл (для финальных событий)
    Timer {
        id: scrollTimer
        interval: 100
        onTriggered: chatFlickable.scrollToBottom()
    }

    // Автоскролл во время стриминга
    Timer {
        id: streamScrollTimer
        interval: 300
        repeat: true
        running: backend.streaming && root.state === "active"
        onTriggered: chatFlickable.scrollToBottom()
    }

    // ── Профиль ──────────────────────────────────────────────────

    function updateProfileName() {
        if (typeof profilesModel !== "undefined" && profilesModel.count > 0) {
            var idx = Math.max(0, Math.min(currentProfileIndex, profilesModel.count - 1))
            currentProfileName = profilesModel.get(idx).name
        } else {
            currentProfileName = "Не выбран"
        }
    }

    Connections {
        target: profilesModel
        onDataChanged: root.updateProfileName()
        onCountChanged: root.updateProfileName()
    }

    onStatusChanged: {
        if (status === PageStatus.Active) updateProfileName()
    }

    Component.onCompleted: updateProfileName()
    allowedOrientations: Orientation.Portrait

    // ── UI ───────────────────────────────────────────────────────

    Rectangle {
        anchors.fill: parent
        color: "white"
        z: -1
    }

    state: (searchInput.activeFocus || chatModel.count > 0 || isSearching) ? "active" : "default"

    Image {
        id: settingsButton
        source: Qt.resolvedUrl("../../images/settings.png")
        width: 36 * root.dP; height: 36 * root.dP
        anchors {
            top: parent.top; right: parent.right
            topMargin: 20 * root.dP; rightMargin: 5 * root.dP
        }

        MouseArea {
            anchors.fill: parent
            onClicked: pageStack.push(Qt.resolvedUrl("ProfilesPage.qml"))
            onPressed: parent.opacity = 0.5
            onReleased: parent.opacity = 1.0
            onCanceled: parent.opacity = 1.0
        }
    }

    Image {
        id: logo
        source: Qt.resolvedUrl("../../images/aseek_logo.png")
        asynchronous: true
        fillMode: Image.PreserveAspectFit
        height: (root.state === "active" ? 29 : 75) * root.dP
        anchors {
            top: parent.top
            horizontalCenter: parent.horizontalCenter
            topMargin: 368.5 * root.dP
        }
        Behavior on height { NumberAnimation { duration: 350; easing.type: Easing.InOutCubic } }
        Behavior on anchors.topMargin { NumberAnimation { duration: 350; easing.type: Easing.InOutCubic } }
    }

    SilicaFlickable {
        id: chatFlickable
        anchors.top: logo.bottom
        anchors.topMargin: 10 * root.dP
        anchors.bottom: statusLabel.visible ? statusLabel.top : profileLabel.top
        anchors.bottomMargin: 10 * root.dP
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.margins: 10 * root.dP
        contentHeight: chatCol.height
        clip: true
        visible: root.state === "active"

        function scrollToBottom() {
            contentY = Math.max(0, contentHeight - height)
        }

        Column {
            id: chatCol
            width: parent.width
            spacing: 12 * root.dP

            Repeater {
                model: chatModel
                delegate: Rectangle {
                    id: bubble
                    property var messageParts: mdConverter.parseToParts(model.text)

                    width: parent.width * 0.85
                    height: msgPartsColumn.height + 30 * root.dP
                    radius: 15 * root.dP
                    color: model.type === "user" ? "#2e67f2" : "#f1f3f5"
                    anchors.right: model.type === "user" ? parent.right : undefined

                    // Курсор-индикатор генерации
                    property bool isStreamingThis:
                        (model.index === root.streamingItemIndex) && backend.streaming

                    Column {
                        id: msgPartsColumn
                        width: parent.width - 30 * root.dP
                        anchors.centerIn: parent
                        spacing: 8 * root.dP

                        Repeater {
                            model: bubble.messageParts

                            delegate: Column {
                                width: msgPartsColumn.width

                                Text {
                                    visible: !modelData.isCode
                                    width: parent.width
                                    height: visible ? implicitHeight : 0
                                    text: !modelData.isCode ? String(modelData.content) : ""
                                    textFormat: Text.RichText
                                    wrapMode: Text.Wrap
                                    font.pixelSize: Theme.fontSizeMedium * root.fontScale
                                    color: bubble.parent && model.type === "user"
                                           ? "white" : "black"
                                }

                                SilicaFlickable {
                                    id: codeBox
                                    visible: modelData.isCode
                                    width: parent.width
                                    height: visible
                                        ? (codeLabel.implicitHeight + 20 * root.dP) : 0
                                    contentWidth: Math.max(
                                        width, codeLabel.implicitWidth + 40 * root.dP)
                                    flickableDirection: Flickable.HorizontalFlick
                                    clip: true

                                    Rectangle {
                                        anchors.fill: parent
                                        color: "#000000"
                                        opacity: 0.1
                                        radius: 6 * root.dP
                                    }

                                    Label {
                                        id: codeLabel
                                        anchors.verticalCenter: parent.verticalCenter
                                        x: 15 * root.dP
                                        text: modelData.isCode
                                            ? String(modelData.content) : ""
                                        font.family: "Monospace"
                                        font.pixelSize:
                                            Theme.fontSizeExtraSmall * root.fontScale
                                        color: "#d63384"
                                    }

                                    HorizontalScrollDecorator { flickable: codeBox }
                                }
                            }
                        }

                        // Мигающий курсор ▌ во время стриминга
                        Text {
                            visible: bubble.isStreamingThis
                            text: "▌"
                            font.pixelSize: Theme.fontSizeMedium * root.fontScale
                            color: "#2e67f2"

                            SequentialAnimation on opacity {
                                running: bubble.isStreamingThis
                                loops: Animation.Infinite
                                NumberAnimation {
                                    from: 1.0; to: 0.0
                                    duration: 500
                                }
                                NumberAnimation {
                                    from: 0.0; to: 1.0
                                    duration: 500
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    // ── Статус бэкенда ───────────────────────────────────────────

    Label {
        id: statusLabel
        text: {
            if (backend.status === "starting")
                return "Запуск бэкенда, загрузка модели..."
            if (backend.streaming)
                return "Генерация ответа..."
            if (backend.status === "busy")
                return "Поиск и подготовка контекста..."
            return ""
        }
        color: "#2e67f2"
        font.pixelSize: 13 * root.dP
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: profileLabel.top
        anchors.bottomMargin: 4 * root.dP
        visible: backend.status === "starting"
                 || backend.status === "busy"
    }

    // ── Индикатор состояния бэкенда (точка) ──────────────────────

    Rectangle {
        id: backendIndicator
        width: 10 * root.dP; height: 10 * root.dP
        radius: 5 * root.dP
        anchors {
            top: parent.top
            left: parent.left
            topMargin: 25 * root.dP
            leftMargin: 10 * root.dP
        }
        color: {
            if (backend.status === "ready") return "#4CAF50"
            if (backend.status === "busy") return "#FF9800"
            if (backend.status === "starting") return "#2196F3"
            return "#F44336"
        }

        // Пульсация точки при стриминге
        SequentialAnimation on scale {
            running: backend.streaming
            loops: Animation.Infinite
            NumberAnimation { from: 1.0; to: 1.3; duration: 400 }
            NumberAnimation { from: 1.3; to: 1.0; duration: 400 }
        }

        Behavior on color { ColorAnimation { duration: 300 } }
    }

    // ── Поле ввода и кнопка отправки ─────────────────────────────

    Row {
        id: searchContainer
        width: 384 * root.dP
        height: 71 * root.dP
        anchors.horizontalCenter: parent.horizontalCenter
        spacing: 7 * root.dP
        y: logo.y + logo.height + (12 * root.dP)

        Rectangle {
            id: inputRect
            width: root.state === "active"
                ? (parent.width - sendButton.width - parent.spacing)
                : parent.width
            height: root.state === "active" ? 55 * root.dP : parent.height
            color: "white"
            border.color: "#2e67f2"; border.width: 2; radius: 10

            TextInput {
                id: searchInput
                anchors.fill: parent; anchors.margins: Theme.paddingMedium
                verticalAlignment: TextInput.AlignVCenter
                color: isSearching ? "#a0a0a0" : "black"
                font.pixelSize: Theme.fontSizeMedium; clip: true
                enabled: !isSearching

                Keys.onReturnPressed: root.startSearch()
                Keys.onEnterPressed: root.startSearch()

                Text {
                    id: placeholder
                    anchors.fill: parent
                    text: {
                        if (backend.status === "starting")
                            return "Загрузка модели..."
                        if (backend.streaming)
                            return "Генерация ответа..."
                        return root.state === "active"
                            ? "Введите вопрос..."
                            : "Воспользуйтесь интеллектуальным поиском Aseek..."
                    }
                    visible: !parent.text && !parent.activeFocus && !isSearching
                    color: "#808080"
                    verticalAlignment: Text.AlignVCenter
                    font.pixelSize: parent.font.pixelSize
                    fontSizeMode: Text.Fit; minimumPixelSize: 10
                }
            }
        }

        Image {
            id: sendButton
            source: isSearching
                ? Qt.resolvedUrl("../../images/stop_icon.png")
                : Qt.resolvedUrl("../../images/send_icon.png")
            width: 55 * root.dP; height: 55 * root.dP
            opacity: root.state === "active" ? 1 : 0
            visible: opacity > 0
            anchors.verticalCenter: inputRect.verticalCenter

            MouseArea {
                anchors.fill: parent
                onClicked: {
                    if (isSearching) {
                        root.stopSearch()
                    } else {
                        root.startSearch()
                    }
                }
                onPressed: parent.scale = 0.9
                onReleased: parent.scale = 1.0
            }
            Behavior on scale { NumberAnimation { duration: 100 } }
        }
    }

    Label {
        id: profileLabel
        text: "Текущий профиль: " + root.currentProfileName
        color: "#a0a0a0"
        font.pixelSize: Theme.fontSizeExtraSmall
        anchors.horizontalCenter: parent.horizontalCenter
        y: root.state === "active"
           ? searchContainer.y - height - Theme.paddingSmall
           : searchContainer.y + searchContainer.height + Theme.paddingSmall
    }

    states: [
        State {
            name: "active"
            AnchorChanges {
                target: logo
                anchors.horizontalCenter: undefined
                anchors.left: parent.left
            }
            PropertyChanges {
                target: logo
                anchors.leftMargin: 5 * root.dP
                anchors.topMargin: 23.5 * root.dP
            }
            PropertyChanges {
                target: searchContainer
                y: parent.height - searchContainer.height - (22 * root.dP)
            }
        }
    ]

    transitions: Transition {
        AnchorAnimation { duration: 350; easing.type: Easing.InOutCubic }
        NumberAnimation {
            properties: "y,x,height,width,opacity,topMargin,leftMargin"
            duration: 350; easing.type: Easing.InOutCubic
        }
    }
}
