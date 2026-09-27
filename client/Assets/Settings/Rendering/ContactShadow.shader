// Soft elliptical contact shadow drawn procedurally on a unit quad.
// No texture: alpha is a radial smoothstep falloff. SRP-Batcher compatible
// (all per-material parameters live in the UnityPerMaterial CBUFFER).
Shader "ThinhThan/ContactShadow"
{
    Properties
    {
        _BaseColor("Shadow Color", Color) = (0, 0, 0, 0.45)
        _Softness("Edge Softness", Range(0.001, 1)) = 0.35
    }

    SubShader
    {
        Tags
        {
            "RenderType" = "Transparent"
            "Queue" = "Transparent"
            "RenderPipeline" = "UniversalPipeline"
            "IgnoreProjector" = "True"
        }

        Pass
        {
            Name "ContactShadow"

            Blend SrcAlpha OneMinusSrcAlpha
            Cull Off
            ZWrite Off
            ZTest LEqual

            HLSLPROGRAM
            #pragma vertex Vert
            #pragma fragment Frag

            #include "Packages/com.unity.render-pipelines.universal/ShaderLibrary/Core.hlsl"

            struct Attributes
            {
                float4 positionOS : POSITION;
                float2 uv : TEXCOORD0;
            };

            struct Varyings
            {
                float4 positionHCS : SV_POSITION;
                float2 uv : TEXCOORD0;
            };

            CBUFFER_START(UnityPerMaterial)
                half4 _BaseColor;
                half _Softness;
            CBUFFER_END

            Varyings Vert(Attributes input)
            {
                Varyings output;
                output.positionHCS = TransformObjectToHClip(input.positionOS.xyz);
                output.uv = input.uv;
                return output;
            }

            half4 Frag(Varyings input) : SV_Target
            {
                float2 centered = input.uv * 2.0 - 1.0;
                float dist = length(centered);
                half alpha = _BaseColor.a * (half)(1.0 - smoothstep(1.0 - _Softness, 1.0, dist));
                return half4(_BaseColor.rgb, alpha);
            }
            ENDHLSL
        }
    }

    Fallback Off
}
