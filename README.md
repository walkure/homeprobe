# 自宅環境観測プローブ

Prometheusで自宅環境を観測するためのExpoter

Goで書いています。(go version go1.21.0 linux/arm)

# センサなど
ハードウェアは Raspberry Pi Zero W を使っていますが、必要なI/Fが実装されていれば他のでもいけそう。
 
- I2C接続(この順でセンサを探します)
  - BME280 [ＢＭＥ２８０使用　温湿度・気圧センサモジュールキット](https://akizukidenshi.com/catalog/g/gK-09421/)
  - CCS811 [CCS811搭載 空気品質センサモジュール](https://www.switch-science.com/catalog/3298/)
  - SHT35 [GROVE - I2C 高精度温湿度センサ（SHT35）](https://www.switch-science.com/catalog/5337/)
- シリアル接続
  - MH-Z19B/C [ＣＯ２センサーモジュール　ＭＨ－Ｚ１９Ｃ](https://akizukidenshi.com/catalog/g/gM-16142/)
- Bluetooth Low Energy(BLE)
  - WxBeacon2(2JCIE-BL01) [WxBeacon2](https://weathernews.jp/smart/wxbeacon2/)
    - EPモード(General/Limited Broadcaster 2)に設定されていることを期待しています。

# ビルド

 `make` で `co2`/`i2cdev`/`wxbeacon2`の3バイナリを作ります。 `./bin`にバイナリを吐くので、`sudo mv ./bin/* /usr/local/bin/`などで。

 GitHub Actionsでarmとarm64のビルドを作って[Release](https://github.com/walkure/homeprobe/releases)に入るようにしてあります。

# 起動設定

`unit` にそれぞれのバイナリを起動するためのsystemd sample unitファイル例を入れてあります。

listenするアドレスはデフォルトで`:9821`ですが、`--listen`で適当に変更して衝突しないようにしてください。

- co2
  - MH-Z19Bへアクセスできるtty deviceのpathを引数`--mhz19`で渡してください。
- i2cdev
  - Raspberry Pi OSの場合、起動ユーザが`i2c`グループメンバである必要があります。
  - BME280の出力する温度情報はどうも数度高めに出るようなので、`--temp_offset`でオフセットを設定できるようにしてあります。
  - 海面更正気圧を記録する場合は`--above_sea_level`に海抜(m)を設定してください。
- wxbeacon2
  - Linuxの場合、BLEの操作に`CAP_NET_ADMIN`が必要です。
  - WxBeacon2のMacアドレスを引数`--wxbeacon`に渡してください。
  - 海面更正気圧を記録する場合は`--above_sea_level`に海抜(m)を設定してください。


# ライセンス
MIT

# 作者
walkure
