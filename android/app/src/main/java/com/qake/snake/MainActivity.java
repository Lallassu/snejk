package com.qake.snake;

import android.app.Activity;
import android.graphics.Color;
import android.os.Bundle;
import android.util.TypedValue;
import android.view.Gravity;
import android.widget.LinearLayout;
import android.widget.TextView;

public class MainActivity extends Activity {
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        root.setGravity(Gravity.CENTER);
        root.setPadding(48, 48, 48, 48);
        root.setBackgroundColor(Color.rgb(18, 22, 29));

        TextView title = new TextView(this);
        title.setText(getString(R.string.app_name));
        title.setTextColor(Color.WHITE);
        title.setTextSize(TypedValue.COMPLEX_UNIT_SP, 28);
        title.setGravity(Gravity.CENTER);

        TextView msg = new TextView(this);
        msg.setText(getString(R.string.bootstrap_message));
        msg.setTextColor(Color.rgb(210, 220, 230));
        msg.setTextSize(TypedValue.COMPLEX_UNIT_SP, 16);
        msg.setGravity(Gravity.CENTER);
        msg.setPadding(0, 24, 0, 0);

        root.addView(title);
        root.addView(msg);

        setContentView(root);
    }
}
