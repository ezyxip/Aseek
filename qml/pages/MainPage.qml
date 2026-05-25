import QtQuick 2.0
import Sailfish.Silica 1.0
import ru.aseek.utils 1.0

Page {
    id: root
    objectName: "mainPage"

    readonly property real dP: Screen.width / 402
    readonly property real fontScale: 0.7

    property string pipelineStage: "idle"
    property string streamingText: ""
    property string errorText: ""
    property int docCount: 0
    property int rerankedCount: 0
    property string lastQuery: ""

    property bool isBusy: pipelineStage !== "idle" && pipelineStage !== "done" && pipelineStage !== "error"

    property int streamingItemIndex: -1

    MDConverter { id: mdConverter }

    ListModel { id: chatModel }

    function stageColor(stage) {
        if (pipelineStage === "error") return "#ff4d4d"
        var order = ["searching", "reranking", "prefill", "streaming"]
        var idx = order.indexOf(stage)
        var curIdx = order.indexOf(pipelineStage)
        if (pipelineStage === "done") return "#4CAF50"
        if (idx < 0) return "#808080"
        if (idx < curIdx) return "#4CAF50"
        if (idx === curIdx) return "#2e67f2"
        return "#d0d0d0"
    }

    function stageDone(stage) {
        var order = ["searching", "reranking", "prefill", "streaming"]
        var idx = order.indexOf(stage)
        var curIdx = order.indexOf(pipelineStage)
        if (pipelineStage === "done") return true
        return idx < curIdx
    }

    function stageActive(stage) {
        var order = ["searching", "reranking", "prefill", "streaming"]
        var idx = order.indexOf(stage)
        var curIdx = order.indexOf(pipelineStage)
        if (pipelineStage === "done" || pipelineStage === "error") return false
        return idx === curIdx
    }

    function stageVisible(stage) {
        var order = ["searching", "reranking", "prefill", "streaming"]
        var idx = order.indexOf(stage)
        var curIdx = order.indexOf(pipelineStage)
        if (pipelineStage === "done") return true
        if (pipelineStage === "error") return idx <= curIdx
        return idx <= curIdx + 1
    }

    function stageDetail(stage) {
        if (stage === "searching") {
            if (pipelineStage === "searching") return qsTr("Поиск...")
            if (stageDone(stage)) return docCount > 0 ? qsTr("Найдено: %1").arg(docCount) : qsTr("Готово")
        }
        if (stage === "reranking") {
            if (pipelineStage === "reranking") return qsTr("Ранжирование...")
            if (stageDone(stage)) return rerankedCount > 0 ? qsTr("Отобрано: %1").arg(rerankedCount) : qsTr("Готово")
        }
        if (stage === "prefill") {
            if (pipelineStage === "prefill") return qsTr("Префилл...")
            if (stageDone(stage)) return qsTr("Готово")
        }
        if (stage === "streaming") {
            if (pipelineStage === "streaming") return qsTr("Генерация...")
            if (stageDone(stage)) return qsTr("Завершено")
        }
        return ""
    }

    function startSearch() {
        if (searchInput.text.trim() === "" || isBusy) return

        var query = searchInput.text.trim()
        lastQuery = query
        streamingText = ""
        errorText = ""
        docCount = 5
        rerankedCount = 3

        chatModel.append({ "type": "user", "text": query })
        chatModel.append({ "type": "bot", "text": "..." })
        streamingItemIndex = chatModel.count - 1

        searchInput.text = ""
        pipelineStage = "searching"
        scrollTimer.start()
        simulatePipeline()
    }

    function stopSearch() {
        var wasStreaming = (streamingItemIndex >= 0)
        var idx = streamingItemIndex
        streamingItemIndex = -1

        if (wasStreaming && idx >= 0 && idx < chatModel.count) {
            var currentText = chatModel.get(idx).text
            if (!currentText || currentText === "...") {
                chatModel.setProperty(idx, "text", "*[Прервано]*")
            } else {
                chatModel.setProperty(idx, "text", currentText + "\n\n*[Прервано]*")
            }
        }
        pipelineStage = "idle"
        streamingText = ""
        errorText = ""
    }

    function simulatePipeline() {
        if (pipelineStage === "error" || pipelineStage === "idle") return

        if (pipelineStage === "searching") {
            queryTimer.interval = 1200
            queryTimer.callback = function() {
                if (pipelineStage !== "searching") return
                pipelineStage = "reranking"
                simulatePipeline()
            }
            queryTimer.start()
        } else if (pipelineStage === "reranking") {
            queryTimer.interval = 800
            queryTimer.callback = function() {
                if (pipelineStage !== "reranking") return
                pipelineStage = "prefill"
                simulatePipeline()
            }
            queryTimer.start()
        } else if (pipelineStage === "prefill") {
            queryTimer.interval = 500
            queryTimer.callback = function() {
                if (pipelineStage !== "prefill") return
                pipelineStage = "streaming"
                scrollTimer.start()
                simulateStreaming()
            }
            queryTimer.start()
        }
    }

    function simulateStreaming() {
        streamingText = ""
        var fullText = "Это пример ответа на ваш запрос. Модель обрабатывает контекст и генерирует релевантный ответ на основе найденных документов.\n\n**Ключевые моменты:**\n- Найдено " + docCount + " релевантных документов\n- Отобрано " + rerankedCount + " для контекста\n- Ответ сгенерирован на устройстве через llama.cpp\n\n```\nПример кода\n```\n\nОтвет завершён."
        var pos = 0
        var streamTimer = Qt.createQmlObject('import QtQuick 2.0; Timer { interval: 30; repeat: true; property var callback; property string text; property int pos: 0; onTriggered: { if (pos < text.length) { callback(text.substring(0, pos + 2)); pos += 2; } else { stop(); callback(text); } } }', root, "streamTimer")
        streamTimer.text = fullText
        streamTimer.callback = function(text) {
            streamingText = text
            if (streamingItemIndex >= 0 && streamingItemIndex < chatModel.count) {
                chatModel.setProperty(streamingItemIndex, "text", text)
            }
            if (text.length >= fullText.length) {
                pipelineStage = "done"
                streamingItemIndex = -1
                scrollTimer.start()
            }
        }
        streamTimer.start()
    }

    Timer {
        id: scrollTimer
        interval: 100
        onTriggered: chatFlickable.scrollToBottom()
    }

    Timer {
        id: streamScrollTimer
        interval: 300
        repeat: true
        running: pipelineStage === "streaming" && root.state === "active"
        onTriggered: chatFlickable.scrollToBottom()
    }

    Timer {
        id: queryTimer
        interval: 1
        repeat: false
        property var callback: function() {}
        onTriggered: callback()
    }

    state: (searchInput.activeFocus || chatModel.count > 0 || isBusy) ? "active" : "default"

    Rectangle {
        anchors.fill: parent
        color: "white"
        z: -1
    }

    Rectangle {
        id: backendIndicator
        width: 10 * dP; height: 10 * dP
        radius: 5 * dP
        anchors {
            top: parent.top; left: parent.left
            topMargin: 25 * dP; leftMargin: 10 * dP
        }
        color: {
            if (pipelineStage === "error") return "#F44336"
            if (isBusy) return "#FF9800"
            if (pipelineStage === "done") return "#4CAF50"
            return "#4CAF50"
        }

        SequentialAnimation on scale {
            running: isBusy
            loops: Animation.Infinite
            NumberAnimation { from: 1.0; to: 1.3; duration: 400 }
            NumberAnimation { from: 1.3; to: 1.0; duration: 400 }
        }

        Behavior on color { ColorAnimation { duration: 300 } }
    }

    Image {
        id: settingsButton
        source: Qt.resolvedUrl("../../images/settings.png")
        width: 36 * dP; height: 36 * dP
        anchors {
            top: parent.top; right: parent.right
            topMargin: 20 * dP; rightMargin: 5 * dP
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
        height: (root.state === "active" ? 29 : 75) * dP
        anchors {
            top: parent.top
            horizontalCenter: parent.horizontalCenter
            topMargin: 368.5 * dP
        }
        Behavior on height { NumberAnimation { duration: 350; easing.type: Easing.InOutCubic } }
        Behavior on anchors.topMargin { NumberAnimation { duration: 350; easing.type: Easing.InOutCubic } }
    }

    SilicaFlickable {
        id: chatFlickable
        anchors.top: logo.bottom
        anchors.topMargin: 10 * dP
        anchors.bottom: pipelineIndicator.visible ? pipelineIndicator.top : statusLabel.visible ? statusLabel.top : profileLabel.top
        anchors.bottomMargin: 10 * dP
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.margins: 10 * dP
        contentHeight: chatCol.height
        clip: true
        visible: root.state === "active"

        function scrollToBottom() {
            contentY = Math.max(0, contentHeight - height)
        }

        Column {
            id: chatCol
            width: parent.width
            spacing: 12 * dP

            Repeater {
                model: chatModel

                delegate: Rectangle {
                    id: bubble
                    property var messageParts: mdConverter.parseToParts(model.text)

                    width: parent.width * 0.85
                    height: msgPartsColumn.height + 30 * dP
                    radius: 15 * dP
                    color: model.type === "user" ? "#2e67f2" : "#f1f3f5"
                    anchors.right: model.type === "user" ? parent.right : undefined

                    property bool isStreamingThis: (model.index === root.streamingItemIndex) && pipelineStage === "streaming"

                    Column {
                        id: msgPartsColumn
                        width: parent.width - 30 * dP
                        anchors.centerIn: parent
                        spacing: 8 * dP

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
                                    color: model.type === "user" ? "white" : "black"
                                }

                                SilicaFlickable {
                                    id: codeBox
                                    visible: modelData.isCode
                                    width: parent.width
                                    height: visible ? (codeLabel.implicitHeight + 20 * dP) : 0
                                    contentWidth: Math.max(width, codeLabel.implicitWidth + 40 * dP)
                                    flickableDirection: Flickable.HorizontalFlick
                                    clip: true

                                    Rectangle {
                                        anchors.fill: parent
                                        color: "#000000"
                                        opacity: 0.1
                                        radius: 6 * dP
                                    }

                                    Label {
                                        id: codeLabel
                                        anchors.verticalCenter: parent.verticalCenter
                                        x: 15 * dP
                                        text: modelData.isCode ? String(modelData.content) : ""
                                        font.family: "Monospace"
                                        font.pixelSize: Theme.fontSizeExtraSmall * root.fontScale
                                        color: "#d63384"
                                    }

                                    HorizontalScrollDecorator { flickable: codeBox }
                                }
                            }
                        }

                        Text {
                            visible: bubble.isStreamingThis
                            text: "\u258C"
                            font.pixelSize: Theme.fontSizeMedium * root.fontScale
                            color: "#2e67f2"

                            SequentialAnimation on opacity {
                                running: bubble.isStreamingThis
                                loops: Animation.Infinite
                                NumberAnimation { from: 1.0; to: 0.0; duration: 500 }
                                NumberAnimation { from: 0.0; to: 1.0; duration: 500 }
                            }
                        }
                    }
                }
            }
        }
    }

    Column {
        id: pipelineIndicator
        visible: pipelineStage !== "idle"
        width: parent.width - 20 * dP
        anchors {
            horizontalCenter: parent.horizontalCenter
            bottom: searchContainer.top
            bottomMargin: 10 * dP
        }
        spacing: 4 * dP

        Repeater {
            model: ["searching", "reranking", "prefill", "streaming"]

            delegate: Item {
                width: parent.width
                height: 24 * dP
                visible: stageVisible(modelData)

                Rectangle {
                    id: dot
                    anchors.verticalCenter: parent.verticalCenter
                    width: 12 * dP; height: 12 * dP
                    radius: width / 2
                    color: stageColor(modelData)
                    opacity: stageDone(modelData) ? 1.0 : (stageActive(modelData) ? 1.0 : 0.4)

                    BusyIndicator {
                        anchors.centerIn: parent
                        size: BusyIndicatorSize.Small
                        running: stageActive(modelData) && pipelineStage !== "prefill"
                        visible: running
                        color: "#2e67f2"
                    }

                    Label {
                        anchors.centerIn: parent
                        text: stageDone(modelData) ? "\u2713" : ""
                        color: "white"
                        font.pixelSize: 8 * dP
                        visible: text !== ""
                    }
                }

                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    anchors.left: dot.right
                    anchors.leftMargin: 6 * dP
                    font.pixelSize: 12 * dP
                    color: stageColor(modelData)
                    text: {
                        switch (modelData) {
                        case "searching": return qsTr("Поиск документов")
                        case "reranking": return qsTr("Ранжирование")
                        case "prefill": return qsTr("Префилл")
                        case "streaming": return qsTr("Генерация")
                        }
                    }
                }

                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    anchors.right: parent.right
                    font.pixelSize: 10 * dP
                    color: "#808080"
                    text: stageDetail(modelData)
                    visible: text !== ""
                }
            }
        }
    }

    Label {
        id: statusLabel
        text: {
            if (pipelineStage === "error") return qsTr("Ошибка")
            if (pipelineStage === "searching" || pipelineStage === "reranking") return qsTr("Поиск и подготовка контекста...")
            if (pipelineStage === "streaming") return qsTr("Генерация ответа...")
            return ""
        }
        color: "#2e67f2"
        font.pixelSize: 13 * dP
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: profileLabel.top
        anchors.bottomMargin: 4 * dP
        visible: pipelineStage !== "idle" && pipelineStage !== "done"
    }

    Label {
        id: profileLabel
        text: qsTr("Текущий профиль: ") + (profilesModel.count > 0 ? profilesModel.get(currentProfileIndex).name : qsTr("Не выбран"))
        color: "#a0a0a0"
        font.pixelSize: Theme.fontSizeExtraSmall
        anchors.horizontalCenter: parent.horizontalCenter
        y: root.state === "active"
           ? searchContainer.y - height - Theme.paddingSmall
           : searchContainer.y + searchContainer.height + Theme.paddingSmall
    }

    Row {
        id: searchContainer
        width: 384 * dP
        height: 71 * dP
        anchors.horizontalCenter: parent.horizontalCenter
        spacing: 7 * dP
        y: logo.y + logo.height + 12 * dP

        Rectangle {
            id: inputRect
            width: root.state === "active"
                ? (parent.width - sendButton.width - parent.spacing)
                : parent.width
            height: root.state === "active" ? 55 * dP : parent.height
            color: "white"
            border.color: "#2e67f2"; border.width: 2; radius: 10

            TextInput {
                id: searchInput
                anchors.fill: parent; anchors.margins: Theme.paddingMedium
                verticalAlignment: TextInput.AlignVCenter
                color: isBusy ? "#a0a0a0" : "black"
                font.pixelSize: Theme.fontSizeMedium; clip: true
                enabled: !isBusy

                Keys.onReturnPressed: root.startSearch()
                Keys.onEnterPressed: root.startSearch()

                Text {
                    anchors.fill: parent
                    text: {
                        if (isBusy) return qsTr("Подождите...")
                        return root.state === "active"
                            ? qsTr("Введите вопрос...")
                            : qsTr("Воспользуйтесь интеллектуальным поиском Aseek...")
                    }
                    visible: !parent.text && !parent.activeFocus && !isBusy
                    color: "#808080"
                    verticalAlignment: Text.AlignVCenter
                    font.pixelSize: parent.font.pixelSize
                    fontSizeMode: Text.Fit; minimumPixelSize: 10
                }
            }
        }

        Image {
            id: sendButton
            source: isBusy
                ? Qt.resolvedUrl("../../images/stop_icon.png")
                : Qt.resolvedUrl("../../images/send_icon.png")
            width: 55 * dP; height: 55 * dP
            opacity: root.state === "active" ? 1 : 0
            visible: opacity > 0
            anchors.verticalCenter: inputRect.verticalCenter

            MouseArea {
                anchors.fill: parent
                onClicked: {
                    if (isBusy) root.stopSearch()
                    else root.startSearch()
                }
                onPressed: parent.scale = 0.9
                onReleased: parent.scale = 1.0
            }
            Behavior on scale { NumberAnimation { duration: 100 } }
        }
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
                anchors.leftMargin: 5 * dP
                anchors.topMargin: 23.5 * dP
            }
            PropertyChanges {
                target: searchContainer
                y: parent.height - searchContainer.height - 22 * dP
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