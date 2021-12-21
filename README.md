# 自宅環境観測プローブ

Prometheusで自宅環境を観測するためのExpoter

Goで書いています。(go version go1.15.3 linux/arm)

# センサなど
ハードウェアは Raspberry Pi Zero W を使っていますが、必要なI/Fが実装されていれば他のでもいけそう。
 
- I2C接続
  - BME280 [ＢＭＥ２８０使用　温湿度・気圧センサモジュールキット](https://akizukidenshi.com/catalog/g/gK-09421/)
  - CCS811 [CCS811搭載 空気品質センサモジュール](https://www.switch-science.com/catalog/3298/)
  - SHT35 [GROVE - I2C 高精度温湿度センサ（SHT35）](https://www.switch-science.com/catalog/5337/)
- シリアル接続
  - MH-Z19B/C [ＣＯ２センサーモジュール　ＭＨ－Ｚ１９Ｃ](https://akizukidenshi.com/catalog/g/gM-16142/)
- Bluetooth Low Energy(BLE)
  - WxBeacon2(2JCIE-BL01) [WxBeacon2](https://weathernews.jp/smart/wxbeacon2/)
    - EPモード(General/Limited Broadcaster 2)に設定されていることを期待しています。

# ビルド
 `go build` 

# 起動設定

WxBeacon2のMacアドレスは最低限必要です。MH-Z19の`tty`など、必要に応じて。

BME280の出力する温度情報はどうも数度高めに出るようなので、適当にオフセットを設定できるようにしてあります。

BLEの操作に`CAP_NET_ADMIN`が必要だったり、I2Cの操作は`i2c`グループメンバじゃないとできないとかあります。systemdで起動するserviceファイルを参照してください。

# ライセンス
MIT

# 作者
walkure